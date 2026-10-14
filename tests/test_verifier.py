"""Each deliberately broken project must be caught in the right stage with the right category."""
import re

from conftest import TRIVIAL_TESTS


def fail(vr):
    f = vr.first_failure
    assert f is not None, "expected a failure but every stage passed"
    return f.stage, f.failure_category


def test_good_project_passes_and_reports_skips(run):
    vr = run()
    assert vr.ok, [s.to_dict() for s in vr.stages if s.status == "failed"]
    assert [s.stage for s in vr.stages] == ["parse", "format", "build", "vet", "test", "probe", "docs", "manifests"]
    assert all(s.status == "passed" for s in vr.stages)


def test_format(run):
    vr = run(lambda f: f.update({"handlers.go": f["handlers.go"].replace("\tmu.Lock()", "      mu.Lock()", 1)}))
    assert fail(vr) == ("format", "format")


def test_compile_error(run):
    vr = run(lambda f: f.update({"handlers.go": f["handlers.go"] + "\nvar x int = \"s\"\n"}))
    assert fail(vr) == ("build", "compile_error")


def test_third_party_import_rejected(run):
    vr = run(lambda f: f.update({"extra.go": 'package main\n\nimport _ "github.com/gorilla/mux"\n'}))
    assert fail(vr) == ("build", "compile_error")
    assert "gorilla" in vr.first_failure.output_excerpt


def test_vet(run):
    vr = run(lambda f: f.update({"extra.go": 'package main\n\nimport "fmt"\n\nfunc vetBad() { fmt.Printf("%d\\n", "x") }\n'}))
    assert fail(vr) == ("vet", "vet")


def test_failing_test(run):
    vr = run(lambda f: f.update({"handlers_test.go": f["handlers_test.go"] + '\nfunc TestBoom(t *testing.T) { t.Fatal("boom") }\n'}))
    assert fail(vr) == ("test", "test_failure")


def test_no_tests(run):
    vr = run(lambda f: f.update({"handlers_test.go": 'package main\n\nimport "testing"\n\nfunc TestOnly(t *testing.T) {}\n'}))
    assert fail(vr) == ("test", "no_tests")


def test_hung_test_times_out(run):
    hang = TRIVIAL_TESTS + '\nfunc TestHang(t *testing.T) { select {} }\n'
    vr = run(lambda f: f.update({"handlers_test.go": hang}), timeouts={"test": 4})
    assert fail(vr) == ("test", "timeout")


def test_missing_metric(run):
    def post(proj):  # drop one metric from the registry the template wrote
        m = proj / "metrics.go"
        m.write_text(re.sub(r".*gpuUtilizationPercent.*\n", "", m.read_text()))
    vr = run(lambda f: f.update({"handlers_test.go": TRIVIAL_TESTS}), post=post)
    assert fail(vr) == ("probe", "probe_metrics")
    assert "gpu_utilization_percent" in vr.first_failure.output_excerpt


def test_wrong_behavior(run):
    vr = run(lambda f: f.update({"handlers.go": f["handlers.go"].replace(' || in["id"] == nil', "")}))
    assert fail(vr) == ("probe", "probe_behavior")


def test_endpoint_missing(run):
    vr = run(lambda f: f.update({"handlers.go": re.sub(r'\tmux.HandleFunc\("GET /jobs".*?\n\t\}\)\n', "", f["handlers.go"], flags=re.S),
                                 "handlers_test.go": TRIVIAL_TESTS}))
    assert fail(vr)[1] in ("probe_behavior", "probe_endpoint")


def test_undocumented_endpoint(run):
    vr = run(lambda f: f.update({"README.md": f["README.md"].replace("- `POST /jobs`\n", "")}))
    assert fail(vr) == ("docs", "docs_mismatch")
    assert "POST /jobs" in vr.first_failure.output_excerpt


def test_documented_but_nonexistent_endpoint(run):
    vr = run(lambda f: f.update({"README.md": f["README.md"] + "\n- `DELETE /ghost`\n"}))
    assert fail(vr) == ("docs", "docs_mismatch")
    assert "/ghost" in vr.first_failure.output_excerpt


def test_undocumented_metric(run):
    vr = run(lambda f: f.update({"README.md": re.sub(r".*gpu_job_queue_length.*\n", "", f["README.md"])}))
    assert fail(vr) == ("docs", "docs_mismatch")


def test_server_that_never_starts(run):
    sleepy = 'package main\n\nimport (\n\t"os"\n\t"time"\n)\n\nfunc init() {\n\tif os.Getenv("PORT") != "" {\n\t\ttime.Sleep(time.Minute)\n\t}\n}\n'
    vr = run(lambda f: f.update({"extra.go": sleepy}), timeouts={"probe": 3})
    assert fail(vr) == ("probe", "probe_health")


def test_hung_handler_is_a_timeout(run):
    def mutate(f):
        f["handlers.go"] = f["handlers.go"].replace('mux.HandleFunc("GET /jobs", func(w http.ResponseWriter, r *http.Request) {',
                                                    'mux.HandleFunc("GET /jobs", func(w http.ResponseWriter, r *http.Request) {\n\t\t<-make(chan int)', 1)
        f["handlers_test.go"] = TRIVIAL_TESTS
    vr = run(mutate, timeouts={"probe": 4})
    assert fail(vr)[1] == "timeout"
