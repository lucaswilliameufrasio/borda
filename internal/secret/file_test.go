package secret

import "testing"

func TestShouldRoundTripSealedPassword(t *testing.T) {
	blob, err := Seal("vault-pass", "router-secret")
	if err != nil {
		t.Fatal(err)
	}
	if string(blob) == "router-secret" || contains(blob, []byte("router-secret")) {
		t.Fatal("sealed file contains the password")
	}
	opened, err := Open("vault-pass", blob)
	if err != nil {
		t.Fatal(err)
	}
	if opened != "router-secret" {
		t.Fatalf("opened %q", opened)
	}
	if _, err := Open("wrong", blob); !errorsIsLocked(err) {
		t.Fatalf("wrong passphrase err %v", err)
	}
}

func contains(blob []byte, needle []byte) bool {
	return len(blob) >= len(needle) && stringIndex(blob, needle) >= 0
}

func stringIndex(blob []byte, needle []byte) int {
	for start := 0; start+len(needle) <= len(blob); start++ {
		matched := true
		for offset := 0; offset < len(needle); offset++ {
			if blob[start+offset] != needle[offset] {
				matched = false
				break
			}
		}
		if matched {
			return start
		}
	}
	return -1
}

func errorsIsLocked(err error) bool {
	return err == ErrLocked
}
