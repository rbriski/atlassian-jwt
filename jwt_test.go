package jwt

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pkg/errors"
)

// Using http://jwt-decoder.herokuapp.com/jwt/decode as a check
var qshTestData = []struct {
	summary   string
	method    string
	url       string
	baseurl   string
	canonical string
	qsh       string
}{
	{
		"No path",
		"GET",
		"http://example.com/",
		"http://example.com",
		"GET&/&",
		"c88caad15a1c1a900b8ac08aa9686f4e8184539bea1deda36e2f649430df3239",
	},
	{
		"Simple path",
		"GET",
		"http://example.com/something",
		"http://example.com",
		"GET&/something&",
		"90b8cc6920375d6a6d133bd65be2d802a20e3544bf5704a1a80151210078798d",
	},
	{
		"Simple query args",
		"GET",
		"http://example.com/something?simple=key",
		"http://example.com",
		"GET&/something&simple=key",
		"3c1ebfedcd634a4146672368a8bee5ae364496a7fabcbda591398c00f03771d2",
	},
	{
		"Simple multiple query args",
		"GET",
		"http://example.com/something?simple=key&another=one",
		"http://example.com",
		"GET&/something&another=one&simple=key",
		"fc039e8c582c0ee90af182a3d29f37ecf1cb368f37b73d5fbd6c5bb66482dda9",
	},
	{
		"Simple post",
		"POST",
		"http://example.com/something?simple=key",
		"http://example.com",
		"POST&/something&simple=key",
		"eafb48e8c78a5b09bc2ce7b0b86289ee4a7bd58179e47006d0141b33d5165af7",
	},
	{
		"JWT on query arg",
		"GET",
		"http://example.com/something?jwt=ABC.DEF.GHI",
		"http://example.com",
		"GET&/something&",
		"90b8cc6920375d6a6d133bd65be2d802a20e3544bf5704a1a80151210078798d",
	},
	{
		"Spaces in the query key",
		"GET",
		"http://example.com/something?some+spaces+in+this+parameter=yes",
		"http://example.com",
		"GET&/something&some%20spaces%20in%20this%20parameter=yes",
		"f73fed24415974607a19ddf9af6f862d9da572ed71bbeb15ae2c64c3aec95317",
	},
	{
		"Non alpha in the query key",
		"GET",
		"http://example.com/something?connect*=yes",
		"http://example.com",
		"GET&/something&connect%2A=yes",
		"4ce3c840740a86e20beafc1d50b106996eaa844de89e879e85c92398cbfb131d",
	},
	{
		"Spaces in the query value",
		"GET",
		"http://example.com/something?param=some+spaces+in+this+parameter",
		"http://example.com",
		"GET&/something&param=some%20spaces%20in%20this%20parameter",
		"c87337655140619583bdaa6f4cc8e67ccd1abba1c1847a67d4dc0cd817e9e38a",
	},
	{
		"Non alpha in the query value",
		"GET",
		"http://example.com/something?param=connect*",
		"http://example.com",
		"GET&/something&param=connect%2A",
		"bff97fd3522d20e35e17076bf8b586924d60c1968ecaca7e6edf5fcbdb53c444",
	},
	{
		"Upcase encoding",
		"GET",
		"http://example.com/something?director=%e5%ae%ae%e5%b4%8e%20%e9%a7%bf",
		"http://example.com",
		"GET&/something&director=%E5%AE%AE%E5%B4%8E%20%E9%A7%BF",
		"459b64741b7a3f0c1fe713a4b796f3a1210402729598179b016fd97aec00b4a8",
	},
	{
		"Sort parameter keys",
		"GET",
		"http://example.com/something?a10=1&a1=2&b1=3&b10=4",
		"http://example.com",
		"GET&/something&a1=2&a10=1&b1=3&b10=4",
		"31e1a2db664e96b08862145f45e290f4045fa894886afbb020e64437dabab07e",
	},
	{
		"Combine repeated parameters",
		"GET",
		"http://example.com/something?tuples=1%2C2%2C3&tuples=6%2C5%2C4&tuples=7%2C9%2C8",
		"http://example.com",
		"GET&/something&tuples=1%2C2%2C3,6%2C5%2C4,7%2C9%2C8",
		"4f6da3fce3772b2b1958f7b97f9f7e8e0919ffc777bb74a892418faedf2fe15b",
	},
	{
		"Combine repeated, unsorted parameters",
		"GET",
		"http://example.com/something?ids=-1&ids=1&ids=20&ids=2&ids=10",
		"http://example.com",
		"GET&/something&ids=-1,1,10,2,20",
		"cfb9607f936f9523743b25897bd3a4ae822536fd18678b984b9dc0b31bfd8c46",
	},
	{
		"with base url",
		"GET",
		"https://example.com/wiki/rest/api/search?cql=title~%22%2A%22+and+type%3Dpage&limit=10",
		"https://example.com/wiki",
		"GET&/rest/api/search&cql=title~%22%2A%22+and+type%3Dpage&limit=10",
		"eb04d45dfbeafaa61923d8ac24b849b593a0c4b82c6f42c0aa0c8546dee33500",
	},
}

// TestQSH is all according to https://developer.atlassian.com/cloud/bitbucket/query-string-hash/
func TestQSH(t *testing.T) {
	for _, data := range qshTestData {
		dummy := &Config{BaseURL: data.baseurl}
		req := httptest.NewRequest(data.method, data.url, nil)
		qsh := dummy.QSH(req)
		if data.qsh != qsh {
			t.Errorf("QSH error [%s], expected %s, got %s", data.summary, data.qsh, qsh)
		}
	}
}

func TestPath(t *testing.T) {
	config := Config{BaseURL: "https://example.com/wiki"}
	req := httptest.NewRequest("GET", "https://example.com/wiki/rest/api/search?cql=title~%22%2A%22+and+type%3Dpage&limit=10", nil)
	path := config.Path(req)
	expectedPath := "/rest/api/search"
	if path != expectedPath {
		t.Errorf("Expected %s, got %s", expectedPath, path)
	}
}

func TestClaimsExpirationAfterIssued(t *testing.T) {
	dummy := &Config{Key: "some_key"}
	claims := dummy.Claims("blah")
	if claims.IssuedAt.After(claims.ExpiresAt.Time) {
		t.Errorf("ExpiredAt should occur after IssuedAt")
	}
}

func TestClaimsIssuerIsKey(t *testing.T) {
	dummy := &Config{Key: "some_key"}
	claims := dummy.Claims("blah")
	if claims.Issuer != dummy.Key {
		t.Errorf("Expected %s, got %s", dummy.Key, claims.Issuer)
	}
}
func TestClaimsQSHIsAdded(t *testing.T) {
	dummy := &Config{Key: "some_key"}
	claims := dummy.Claims("blah")
	if claims.QSH == "" {
		t.Errorf("Expected QSH to be added to claims")
	}
}

func TestClaimsQSHIsCorrect(t *testing.T) {
	dummy := &Config{Key: "some_key"}
	claims := dummy.Claims("blah")
	if claims.QSH != "blah" {
		t.Errorf("Expected %s, got %s", "blah", claims.QSH)
	}
}

func TestAuthHeaderIsSet(t *testing.T) {
	dummy := &Config{
		Key:          "some_key",
		SharedSecret: "some_shared_secret",
	}
	req := httptest.NewRequest("GET", "https://example.com", nil)
	dummy.SetAuthHeader(req)
	if req.Header.Get("Authorization") == "" {
		t.Errorf("Expected Authorization header to be set")
	}
}

type DummyConfig struct {
	Key string
}

func (d *DummyConfig) SetAuthHeader(req *http.Request) error {
	u := req.URL
	if u.Path == "/error" {
		return errors.New("This is an error")
	}

	req.Header.Set("Authorization", "test")
	return nil
}

func TestTransportNilConfig(t *testing.T) {
	tr := &Transport{}

	handler := func(w http.ResponseWriter, r *http.Request) {}
	srv := httptest.NewServer(http.HandlerFunc(handler))
	defer srv.Close()

	client := &http.Client{Transport: tr}
	resp, err := client.Get(srv.URL)
	if err == nil {
		t.Errorf("got no errors, want an error with nil token source")
	}
	if resp != nil {
		t.Errorf("Response = %v; want nil", resp)
	}
}

func TestTransportSetsAuthHeader(t *testing.T) {
	tr := &Transport{
		Config: &DummyConfig{},
	}

	handler := func(w http.ResponseWriter, r *http.Request) {}
	srv := httptest.NewServer(http.HandlerFunc(handler))
	defer srv.Close()

	client := &http.Client{Transport: tr}
	resp, _ := client.Get(srv.URL)
	req := resp.Request
	if req.Header.Get("Authorization") == "" {
		t.Errorf("Authorization header was not set.")
	}
}

func TestTransportUsesBase(t *testing.T) {
	// default transport
	tp := &Transport{}
	if tp.base() != http.DefaultTransport {
		t.Errorf("Expected http.DefaultTransport to be used.")
	}

	// custom transport
	tp = &Transport{
		Base: &http.Transport{},
	}
	if tp.base() == http.DefaultTransport {
		t.Errorf("Expected custom transport to be used.")
	}
}
func TestTransportUsesDefaultOnNil(t *testing.T) {}
