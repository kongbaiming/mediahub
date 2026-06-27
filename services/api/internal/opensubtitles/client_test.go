package opensubtitles

import "testing"

func TestNormalizeIMDBID(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"tt0468569", "tt0468569"},
		{"468569", "tt0468569"},
		{" 468569 ", "tt0468569"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := NormalizeIMDBID(tt.in); got != tt.want {
			t.Errorf("NormalizeIMDBID(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestParseCheckMovieHash2(t *testing.T) {
	sample := `<?xml version="1.0"?>
<methodResponse>
  <params>
    <param>
      <value>
        <struct>
          <member><name>data</name><value><array><data>
            <value><struct>
              <member><name>MovieImdbID</name><value><string>tt0468569</string></value></member>
              <member><name>MovieName</name><value><string>The Dark Knight</string></value></member>
              <member><name>MovieYear</name><value><string>2008</string></value></member>
              <member><name>MovieKind</name><value><string>movie</string></value></member>
            </struct></value>
          </data></array></value></member>
        </struct>
      </value>
    </param>
  </params>
</methodResponse>`
	m, err := parseCheckMovieHash2([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	if m == nil {
		t.Fatal("expected match")
	}
	if m.IMDBID != "tt0468569" {
		t.Errorf("imdb = %q", m.IMDBID)
	}
	if m.Title != "The Dark Knight" {
		t.Errorf("title = %q", m.Title)
	}
	if m.Year != 2008 {
		t.Errorf("year = %d", m.Year)
	}
}
