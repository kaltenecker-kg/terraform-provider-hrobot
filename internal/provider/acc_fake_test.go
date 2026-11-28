package provider

// fakeRobot is an in-memory stand-in for the Hetzner Robot webservice, used
// by the acceptance tests (acc_*_test.go). It speaks the wire format hrobot-go
// expects: form-encoded requests, JSON responses wrapped in a single-key
// envelope ({"key": {...}}, [{"snapshot": {...}}]), and the documented error
// envelope. It models only the endpoints the provider's resources call, and
// it reproduces the API behaviours the provider has to cope with:
//
//   - SSH key data is stored trimmed (the API drops a trailing newline).
//   - A sub-account update leaves fields that are not sent unchanged.
//   - Version-less firewall output rules are stored expanded into separate
//     ipv4 and ipv6 rules.
//
// Tests seed objects that are ordered outside Terraform (servers, failover
// IPs, Storage Boxes) directly, and can make the next matching request fail
// with failNext to exercise partial-failure paths.

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
)

const (
	fakeUser     = "test-user"
	fakePassword = "test-password"
)

type fakeRobot struct {
	t   *testing.T
	srv *httptest.Server

	mu         sync.Mutex
	keys       map[string]map[string]any // fingerprint -> key object
	rdns       map[string]string         // ip -> ptr
	failovers  map[string]*fakeFailover  // ip -> failover
	firewalls  map[int]*fakeFirewall     // server number -> firewall
	vswitches  map[int]*fakeVSwitch      // id -> vswitch
	nextVSwID  int
	boxes      map[int]*fakeBox // storage box id -> box
	failures   map[string]int   // "METHOD /path" -> HTTP status for the next matching request
	subCounter int
}

type fakeFailover struct {
	IP, Netmask, ServerIP string
	ServerNumber          int
	Active                *string
}

type fakeFirewall struct {
	Status       string
	FilterIPv6   bool
	WhitelistHOS bool
	Input        []map[string]string
	Output       []map[string]string
}

type fakeVSwitch struct {
	ID        int
	Name      string
	VLAN      int
	Cancelled bool
	Servers   map[int]bool
}

type fakeBox struct {
	Subs      []map[string]any
	Snapshots []map[string]any
	Plan      map[string]any
	snapSeq   int
}

func newFakeRobot(t *testing.T) *fakeRobot {
	t.Helper()
	f := &fakeRobot{
		t:         t,
		keys:      map[string]map[string]any{},
		rdns:      map[string]string{},
		failovers: map[string]*fakeFailover{},
		firewalls: map[int]*fakeFirewall{},
		vswitches: map[int]*fakeVSwitch{},
		nextVSwID: 50000,
		boxes:     map[int]*fakeBox{},
		failures:  map[string]int{},
	}
	f.srv = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	return f
}

// providerConfig returns a provider block pointing at the fake.
func (f *fakeRobot) providerConfig() string {
	return fmt.Sprintf(`
provider "hrobot" {
  username = %q
  password = %q
  base_url = %q
}
`, fakeUser, fakePassword, f.srv.URL)
}

// --- seeding and inspection helpers (all lock the fake) ---

func (f *fakeRobot) addServer(number int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.firewalls[number] = &fakeFirewall{Status: "disabled"}
}

func (f *fakeRobot) removeServer(number int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.firewalls, number)
}

func (f *fakeRobot) addFailover(ip string, serverNumber int, serverIP string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failovers[ip] = &fakeFailover{IP: ip, Netmask: "255.255.255.255", ServerIP: serverIP, ServerNumber: serverNumber}
}

func (f *fakeRobot) addBox(id int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.boxes[id] = &fakeBox{Plan: map[string]any{
		"status": "disabled", "minute": nil, "hour": nil, "day_of_week": nil,
		"day_of_month": nil, "month": nil, "max_snapshots": 0,
	}}
}

// failNext makes the next request with this method and exact path fail with
// a 400 INVALID_INPUT error (400 is not retried by the client).
func (f *fakeRobot) failNext(method, path string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failures[method+" "+path] = http.StatusBadRequest
}

func (f *fakeRobot) with(fn func()) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fn()
}

// --- HTTP plumbing ---

func (f *fakeRobot) serve(w http.ResponseWriter, r *http.Request) {
	if u, p, ok := r.BasicAuth(); !ok || u != fakeUser || p != fakePassword {
		writeErr(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid credentials")
		return
	}
	body, _ := io.ReadAll(r.Body)
	form, err := url.ParseQuery(string(body))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if status, ok := f.failures[r.Method+" "+r.URL.Path]; ok {
		delete(f.failures, r.Method+" "+r.URL.Path)
		writeErr(w, status, "INVALID_INPUT", "injected failure")
		return
	}

	for _, rt := range f.routes() {
		if rt.method != r.Method {
			continue
		}
		if m := rt.pattern.FindStringSubmatch(r.URL.Path); m != nil {
			params := make([]string, len(m)-1)
			for i, s := range m[1:] {
				params[i], _ = url.PathUnescape(s)
			}
			rt.handle(w, form, params)
			return
		}
	}
	writeErr(w, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("no fake route for %s %s", r.Method, r.URL.Path))
}

type fakeRoute struct {
	method  string
	pattern *regexp.Regexp
	handle  func(w http.ResponseWriter, form url.Values, params []string)
}

func route(method, pattern string, h func(http.ResponseWriter, url.Values, []string)) fakeRoute {
	return fakeRoute{method: method, pattern: regexp.MustCompile("^" + pattern + "$"), handle: h}
}

func (f *fakeRobot) routes() []fakeRoute {
	seg := `([^/]+)`
	num := `(\d+)`
	return []fakeRoute{
		route("GET", "/key", f.keyList),
		route("POST", "/key", f.keyCreate),
		route("GET", "/key/"+seg, f.keyGet),
		route("POST", "/key/"+seg, f.keyRename),
		route("DELETE", "/key/"+seg, f.keyDelete),

		route("GET", "/rdns/"+seg, f.rdnsGet),
		route("PUT", "/rdns/"+seg, f.rdnsSet),
		route("POST", "/rdns/"+seg, f.rdnsSet),
		route("DELETE", "/rdns/"+seg, f.rdnsDelete),

		route("GET", "/failover/"+seg, f.failoverGet),
		route("POST", "/failover/"+seg, f.failoverRoute),
		route("DELETE", "/failover/"+seg, f.failoverUnroute),

		route("GET", "/firewall/"+num, f.firewallGet),
		route("POST", "/firewall/"+num, f.firewallUpdate),
		route("DELETE", "/firewall/"+num, f.firewallDelete),

		route("POST", "/vswitch", f.vswitchCreate),
		route("GET", "/vswitch/"+num, f.vswitchGet),
		route("POST", "/vswitch/"+num, f.vswitchUpdate),
		route("DELETE", "/vswitch/"+num, f.vswitchDelete),
		route("POST", "/vswitch/"+num+"/server", f.vswitchAddServers),
		route("DELETE", "/vswitch/"+num+"/server", f.vswitchRemoveServers),

		route("GET", "/storagebox/"+num+"/subaccount", f.subList),
		route("POST", "/storagebox/"+num+"/subaccount", f.subCreate),
		route("PUT", "/storagebox/"+num+"/subaccount/"+seg, f.subUpdate),
		route("DELETE", "/storagebox/"+num+"/subaccount/"+seg, f.subDelete),

		route("GET", "/storagebox/"+num+"/snapshot", f.snapList),
		route("POST", "/storagebox/"+num+"/snapshot", f.snapCreate),
		route("DELETE", "/storagebox/"+num+"/snapshot/"+seg, f.snapDelete),
		route("POST", "/storagebox/"+num+"/snapshot/"+seg+"/comment", f.snapComment),

		route("GET", "/storagebox/"+num+"/snapshotplan", f.planGet),
		route("POST", "/storagebox/"+num+"/snapshotplan", f.planSet),
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]any{"error": map[string]any{"status": status, "code": code, "message": msg}})
}

func wrap(key string, v any) map[string]any { return map[string]any{key: v} }

func wrapAll[T any](key string, vs []T) []map[string]any {
	out := make([]map[string]any, len(vs))
	for i, v := range vs {
		out[i] = wrap(key, v)
	}
	return out
}

func atoi(s string) int { n, _ := strconv.Atoi(s); return n }

// --- SSH keys ---

func (f *fakeRobot) keyList(w http.ResponseWriter, _ url.Values, _ []string) {
	fps := make([]string, 0, len(f.keys))
	for fp := range f.keys {
		fps = append(fps, fp)
	}
	sort.Strings(fps)
	list := make([]map[string]any, len(fps))
	for i, fp := range fps {
		list[i] = f.keys[fp]
	}
	writeJSON(w, http.StatusOK, wrapAll("key", list))
}

func (f *fakeRobot) keyCreate(w http.ResponseWriter, form url.Values, _ []string) {
	data := strings.TrimSpace(form.Get("data"))
	sum := md5.Sum([]byte(data))
	hex := fmt.Sprintf("%x", sum)
	pairs := make([]string, 0, len(hex)/2)
	for i := 0; i < len(hex); i += 2 {
		pairs = append(pairs, hex[i:i+2])
	}
	fp := strings.Join(pairs, ":")
	if _, ok := f.keys[fp]; ok {
		writeErr(w, http.StatusConflict, "KEY_ALREADY_EXISTS", "key already exists")
		return
	}
	k := map[string]any{
		"name": form.Get("name"), "fingerprint": fp, "type": "ED25519", "size": 256,
		"data": data, "created_at": "2026-11-28 10:00:00",
	}
	f.keys[fp] = k
	writeJSON(w, http.StatusCreated, wrap("key", k))
}

func (f *fakeRobot) keyGet(w http.ResponseWriter, _ url.Values, p []string) {
	k, ok := f.keys[p[0]]
	if !ok {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", "key not found")
		return
	}
	writeJSON(w, http.StatusOK, wrap("key", k))
}

func (f *fakeRobot) keyRename(w http.ResponseWriter, form url.Values, p []string) {
	k, ok := f.keys[p[0]]
	if !ok {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", "key not found")
		return
	}
	k["name"] = form.Get("name")
	writeJSON(w, http.StatusOK, wrap("key", k))
}

func (f *fakeRobot) keyDelete(w http.ResponseWriter, _ url.Values, p []string) {
	if _, ok := f.keys[p[0]]; !ok {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", "key not found")
		return
	}
	delete(f.keys, p[0])
	w.WriteHeader(http.StatusOK)
}

// --- reverse DNS ---

func (f *fakeRobot) rdnsGet(w http.ResponseWriter, _ url.Values, p []string) {
	ptr, ok := f.rdns[p[0]]
	if !ok {
		writeErr(w, http.StatusNotFound, "RDNS_NOT_FOUND", "rdns entry not found")
		return
	}
	writeJSON(w, http.StatusOK, wrap("rdns", map[string]any{"ip": p[0], "ptr": ptr}))
}

func (f *fakeRobot) rdnsSet(w http.ResponseWriter, form url.Values, p []string) {
	f.rdns[p[0]] = form.Get("ptr")
	writeJSON(w, http.StatusOK, wrap("rdns", map[string]any{"ip": p[0], "ptr": f.rdns[p[0]]}))
}

func (f *fakeRobot) rdnsDelete(w http.ResponseWriter, _ url.Values, p []string) {
	if _, ok := f.rdns[p[0]]; !ok {
		writeErr(w, http.StatusNotFound, "RDNS_NOT_FOUND", "rdns entry not found")
		return
	}
	delete(f.rdns, p[0])
	w.WriteHeader(http.StatusOK)
}

// --- failover IPs ---

func (fo *fakeFailover) json() map[string]any {
	return map[string]any{
		"ip": fo.IP, "netmask": fo.Netmask, "server_ip": fo.ServerIP, "server_ipv6_net": "",
		"server_number": fo.ServerNumber, "active_server_ip": fo.Active,
	}
}

func (f *fakeRobot) failoverGet(w http.ResponseWriter, _ url.Values, p []string) {
	fo, ok := f.failovers[p[0]]
	if !ok {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", "failover ip not found")
		return
	}
	writeJSON(w, http.StatusOK, wrap("failover", fo.json()))
}

func (f *fakeRobot) failoverRoute(w http.ResponseWriter, form url.Values, p []string) {
	fo, ok := f.failovers[p[0]]
	if !ok {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", "failover ip not found")
		return
	}
	active := form.Get("active_server_ip")
	fo.Active = &active
	writeJSON(w, http.StatusOK, wrap("failover", fo.json()))
}

func (f *fakeRobot) failoverUnroute(w http.ResponseWriter, _ url.Values, p []string) {
	fo, ok := f.failovers[p[0]]
	if !ok {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", "failover ip not found")
		return
	}
	fo.Active = nil
	writeJSON(w, http.StatusOK, wrap("failover", fo.json()))
}

// --- firewall ---

var firewallRuleKey = regexp.MustCompile(`^rules\[(input|output)\]\[(\d+)\]\[(\w+)\]$`)

func (fw *fakeFirewall) json(server int) map[string]any {
	rules := func(in []map[string]string) []map[string]string {
		if in == nil {
			return []map[string]string{}
		}
		return in
	}
	return map[string]any{
		"server_ip": fmt.Sprintf("192.0.2.%d", server%250), "server_number": server,
		"status": fw.Status, "filter_ipv6": fw.FilterIPv6, "whitelist_hos": fw.WhitelistHOS,
		"port":  "main",
		"rules": map[string]any{"input": rules(fw.Input), "output": rules(fw.Output)},
	}
}

func (f *fakeRobot) firewallGet(w http.ResponseWriter, _ url.Values, p []string) {
	n := atoi(p[0])
	fw, ok := f.firewalls[n]
	if !ok {
		writeErr(w, http.StatusNotFound, "SERVER_NOT_FOUND", "server not found")
		return
	}
	writeJSON(w, http.StatusOK, wrap("firewall", fw.json(n)))
}

func (f *fakeRobot) firewallUpdate(w http.ResponseWriter, form url.Values, p []string) {
	n := atoi(p[0])
	fw, ok := f.firewalls[n]
	if !ok {
		writeErr(w, http.StatusNotFound, "SERVER_NOT_FOUND", "server not found")
		return
	}
	parsed := map[string]map[int]map[string]string{"input": {}, "output": {}}
	for k, v := range form {
		m := firewallRuleKey.FindStringSubmatch(k)
		if m == nil {
			continue
		}
		idx := atoi(m[2])
		if parsed[m[1]][idx] == nil {
			parsed[m[1]][idx] = map[string]string{}
		}
		parsed[m[1]][idx][m[3]] = v[0]
	}
	ordered := func(dir string) []map[string]string {
		var out []map[string]string
		for i := 0; i < len(parsed[dir]); i++ {
			out = append(out, parsed[dir][i])
		}
		return out
	}
	fw.Status = form.Get("status")
	fw.FilterIPv6 = form.Get("filter_ipv6") == "true"
	fw.WhitelistHOS = form.Get("whitelist_hos") == "true"
	fw.Input = ordered("input")
	// Like the real API, store a version-less output rule as an ipv4 and an
	// ipv6 copy.
	fw.Output = nil
	for _, r := range ordered("output") {
		if r["ip_version"] != "" {
			fw.Output = append(fw.Output, r)
			continue
		}
		for _, v := range []string{"ipv4", "ipv6"} {
			c := map[string]string{}
			for k, val := range r {
				c[k] = val
			}
			c["ip_version"] = v
			fw.Output = append(fw.Output, c)
		}
	}
	writeJSON(w, http.StatusOK, wrap("firewall", fw.json(n)))
}

func (f *fakeRobot) firewallDelete(w http.ResponseWriter, _ url.Values, p []string) {
	n := atoi(p[0])
	if _, ok := f.firewalls[n]; !ok {
		writeErr(w, http.StatusNotFound, "SERVER_NOT_FOUND", "server not found")
		return
	}
	f.firewalls[n] = &fakeFirewall{Status: "disabled"}
	writeJSON(w, http.StatusOK, wrap("firewall", f.firewalls[n].json(n)))
}

// --- vSwitch ---

func (vs *fakeVSwitch) json() map[string]any {
	nums := make([]int, 0, len(vs.Servers))
	for n := range vs.Servers {
		nums = append(nums, n)
	}
	sort.Ints(nums)
	servers := make([]map[string]any, len(nums))
	for i, n := range nums {
		servers[i] = map[string]any{"server_ip": "192.0.2.1", "server_ipv6_net": "", "server_number": n, "status": "ready"}
	}
	return map[string]any{
		"id": vs.ID, "name": vs.Name, "vlan": vs.VLAN, "cancelled": vs.Cancelled,
		"server": servers, "subnet": []any{}, "cloud_network": []any{},
	}
}

func (f *fakeRobot) vswitchCreate(w http.ResponseWriter, form url.Values, _ []string) {
	f.nextVSwID++
	vs := &fakeVSwitch{ID: f.nextVSwID, Name: form.Get("name"), VLAN: atoi(form.Get("vlan")), Servers: map[int]bool{}}
	f.vswitches[vs.ID] = vs
	writeJSON(w, http.StatusCreated, vs.json())
}

func (f *fakeRobot) vswitch(w http.ResponseWriter, id string) *fakeVSwitch {
	vs, ok := f.vswitches[atoi(id)]
	if !ok {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", "vswitch not found")
	}
	return vs
}

func (f *fakeRobot) vswitchGet(w http.ResponseWriter, _ url.Values, p []string) {
	if vs := f.vswitch(w, p[0]); vs != nil {
		writeJSON(w, http.StatusOK, vs.json())
	}
}

func (f *fakeRobot) vswitchUpdate(w http.ResponseWriter, form url.Values, p []string) {
	if vs := f.vswitch(w, p[0]); vs != nil {
		vs.Name = form.Get("name")
		vs.VLAN = atoi(form.Get("vlan"))
		w.WriteHeader(http.StatusCreated)
	}
}

func (f *fakeRobot) vswitchDelete(w http.ResponseWriter, form url.Values, p []string) {
	vs := f.vswitch(w, p[0])
	if vs == nil {
		return
	}
	if form.Get("cancellation_date") == "now" {
		delete(f.vswitches, vs.ID)
	} else {
		vs.Cancelled = true
	}
	w.WriteHeader(http.StatusOK)
}

func (f *fakeRobot) vswitchAddServers(w http.ResponseWriter, form url.Values, p []string) {
	if vs := f.vswitch(w, p[0]); vs != nil {
		for _, s := range form["server[]"] {
			vs.Servers[atoi(s)] = true
		}
		w.WriteHeader(http.StatusCreated)
	}
}

func (f *fakeRobot) vswitchRemoveServers(w http.ResponseWriter, form url.Values, p []string) {
	if vs := f.vswitch(w, p[0]); vs != nil {
		for _, s := range form["server[]"] {
			delete(vs.Servers, atoi(s))
		}
		w.WriteHeader(http.StatusOK)
	}
}

// --- Storage Box ---

func (f *fakeRobot) box(w http.ResponseWriter, id string) *fakeBox {
	b, ok := f.boxes[atoi(id)]
	if !ok {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", "storage box not found")
	}
	return b
}

func setBool(m map[string]any, form url.Values, key string) {
	if v, ok := form[key]; ok {
		m[key] = v[0] == "true"
	}
}

func (f *fakeRobot) subList(w http.ResponseWriter, _ url.Values, p []string) {
	if b := f.box(w, p[0]); b != nil {
		writeJSON(w, http.StatusOK, wrapAll("subaccount", b.Subs))
	}
}

func (f *fakeRobot) subCreate(w http.ResponseWriter, form url.Values, p []string) {
	b := f.box(w, p[0])
	if b == nil {
		return
	}
	f.subCounter++
	user := fmt.Sprintf("u%s-sub%d", p[0], f.subCounter)
	s := map[string]any{
		"username": user, "accountid": fmt.Sprintf("u%s", p[0]), "server": fmt.Sprintf("u%s.your-storagebox.de", p[0]),
		"homedirectory": form.Get("homedirectory"), "samba": false, "ssh": false,
		"external_reachability": false, "webdav": false, "readonly": false,
		"createtime": "2026-11-28 10:00:00", "comment": form.Get("comment"),
	}
	for _, k := range []string{"samba", "ssh", "external_reachability", "webdav", "readonly"} {
		setBool(s, form, k)
	}
	b.Subs = append(b.Subs, s)
	writeJSON(w, http.StatusCreated, wrap("subaccount", map[string]any{
		"username": user, "password": "generated-secret", "accountid": s["accountid"],
		"server": s["server"], "homedirectory": s["homedirectory"],
	}))
}

func (b *fakeBox) sub(user string) (int, map[string]any) {
	for i, s := range b.Subs {
		if s["username"] == user {
			return i, s
		}
	}
	return -1, nil
}

func (f *fakeRobot) subUpdate(w http.ResponseWriter, form url.Values, p []string) {
	b := f.box(w, p[0])
	if b == nil {
		return
	}
	_, s := b.sub(p[1])
	if s == nil {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", "sub-account not found")
		return
	}
	// Fields that are not sent stay as they are.
	for _, k := range []string{"homedirectory", "comment"} {
		if v, ok := form[k]; ok {
			s[k] = v[0]
		}
	}
	for _, k := range []string{"samba", "ssh", "external_reachability", "webdav", "readonly"} {
		setBool(s, form, k)
	}
	w.WriteHeader(http.StatusOK)
}

func (f *fakeRobot) subDelete(w http.ResponseWriter, _ url.Values, p []string) {
	b := f.box(w, p[0])
	if b == nil {
		return
	}
	i, _ := b.sub(p[1])
	if i < 0 {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", "sub-account not found")
		return
	}
	b.Subs = append(b.Subs[:i], b.Subs[i+1:]...)
	w.WriteHeader(http.StatusOK)
}

func (f *fakeRobot) snapList(w http.ResponseWriter, _ url.Values, p []string) {
	if b := f.box(w, p[0]); b != nil {
		writeJSON(w, http.StatusOK, wrapAll("snapshot", b.Snapshots))
	}
}

func (f *fakeRobot) snapCreate(w http.ResponseWriter, _ url.Values, p []string) {
	b := f.box(w, p[0])
	if b == nil {
		return
	}
	b.snapSeq++
	s := map[string]any{
		"name": fmt.Sprintf("2026-11-28T10-00-%02d", b.snapSeq), "timestamp": "2026-11-28T10:00:00+01:00",
		"size": 1024, "filesystem_size": 4096, "automatic": false, "comment": "",
	}
	b.Snapshots = append(b.Snapshots, s)
	writeJSON(w, http.StatusCreated, wrap("snapshot", s))
}

func (b *fakeBox) snapshot(name string) (int, map[string]any) {
	for i, s := range b.Snapshots {
		if s["name"] == name {
			return i, s
		}
	}
	return -1, nil
}

func (f *fakeRobot) snapDelete(w http.ResponseWriter, _ url.Values, p []string) {
	b := f.box(w, p[0])
	if b == nil {
		return
	}
	i, _ := b.snapshot(p[1])
	if i < 0 {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", "snapshot not found")
		return
	}
	b.Snapshots = append(b.Snapshots[:i], b.Snapshots[i+1:]...)
	w.WriteHeader(http.StatusOK)
}

func (f *fakeRobot) snapComment(w http.ResponseWriter, form url.Values, p []string) {
	b := f.box(w, p[0])
	if b == nil {
		return
	}
	_, s := b.snapshot(p[1])
	if s == nil {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", "snapshot not found")
		return
	}
	s["comment"] = form.Get("comment")
	w.WriteHeader(http.StatusOK)
}

func (f *fakeRobot) planGet(w http.ResponseWriter, _ url.Values, p []string) {
	if b := f.box(w, p[0]); b != nil {
		writeJSON(w, http.StatusOK, []map[string]any{wrap("snapshotplan", b.Plan)})
	}
}

func (f *fakeRobot) planSet(w http.ResponseWriter, form url.Values, p []string) {
	b := f.box(w, p[0])
	if b == nil {
		return
	}
	plan := map[string]any{"status": form.Get("status"), "max_snapshots": atoi(form.Get("max_snapshots"))}
	for _, k := range []string{"minute", "hour", "day_of_week", "day_of_month", "month"} {
		if v, ok := form[k]; ok {
			plan[k] = atoi(v[0])
		} else {
			plan[k] = nil
		}
	}
	b.Plan = plan
	writeJSON(w, http.StatusOK, []map[string]any{wrap("snapshotplan", plan)})
}
