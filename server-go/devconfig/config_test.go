package devconfig

import "testing"

func TestParseConfig(t *testing.T) {
	c, err := Parse(`
# the sprite
url = https://chiron.example
key=op://<vault>/<item>/password
op_account = flyio
`)
	if err != nil {
		t.Fatal(err)
	}
	if c.URL != "https://chiron.example" || c.Key != "op://<vault>/<item>/password" || c.OpAccount != "flyio" {
		t.Fatalf("got %+v", c)
	}
}

func TestParseConfigRejectsUnknownAndMalformed(t *testing.T) {
	if _, err := Parse("host = x\n"); err == nil {
		t.Fatal("unknown setting accepted")
	}
	if _, err := Parse("just words\n"); err == nil {
		t.Fatal("malformed line accepted")
	}
}
