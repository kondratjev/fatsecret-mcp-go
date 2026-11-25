package auth

import "testing"

// RFC 5849 §3.4.1 example.
func TestBuildBaseStringRFC5849(t *testing.T) {
	params := map[string]string{
		"file":                   "vacation.jpg",
		"oauth_consumer_key":     "dpf43f3p2l4k3l03",
		"oauth_nonce":            "kllo9940pd9333jh",
		"oauth_signature_method": "HMAC-SHA1",
		"oauth_timestamp":        "1191242096",
		"oauth_token":            "nnch734d00sl2jdk",
		"oauth_version":          "1.0",
		"size":                   "original",
	}
	got := buildBaseString("GET", "http://photos.example.net/photos", params)
	want := "GET&http%3A%2F%2Fphotos.example.net%2Fphotos&" +
		"file%3Dvacation.jpg%26oauth_consumer_key%3Ddpf43f3p2l4k3l03%26" +
		"oauth_nonce%3Dkllo9940pd9333jh%26oauth_signature_method%3DHMAC-SHA1%26" +
		"oauth_timestamp%3D1191242096%26oauth_token%3Dnnch734d00sl2jdk%26" +
		"oauth_version%3D1.0%26size%3Doriginal"
	if got != want {
		t.Fatalf("base string mismatch:\n got: %s\nwant: %s", got, want)
	}
	sig := signBaseString(got, "kd94hf93k423kf44", "pfkkdhi9sl3r4s00")
	if sig != "tR3+Ty81lMeYAr/Fid0kMTYa/WM=" {
		t.Fatalf("signature mismatch: %s", sig)
	}
}

func TestPercentEncode(t *testing.T) {
	if got := percentEncode("a b!~"); got != "a%20b%21~" {
		t.Fatalf("got %q", got)
	}
}
