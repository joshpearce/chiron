package sources

import (
	"context"
	"testing"
)

func TestSearchAsksTheCataloguesAndDerivesRecipes(t *testing.T) {
	srv := fakeHost(t, map[string]string{
		"/opentextbooks/textbooks.json?q=bayesian":                                                 `{"data":[{"id":"288","title":"Think Bayes: Bayesian Statistics Made Simple","license":"Attribution-NonCommercial","description":"Bayesian statistics using computational methods.","contributors":[{"contribution":"Author","first_name":"Allen","last_name":"Downey"}],"formats":[{"type":"Online","url":"https://www.greenteapress.com/thinkbayes/"}],"url":"https://open.umn.edu/opentextbooks/textbooks/think-bayes"}]}`,
		"/api/v1/learning_resources_search/?q=bayesian&platform=ocw&resource_type=course&limit=10": `{"count":1,"results":[{"readable_id":"18.05+spring_2022","title":"Introduction to Probability and Statistics","url":"https://ocw.mit.edu/courses/18-05-introduction-to-probability-and-statistics-spring-2022/","description":"<p>Probability and Bayesian inference.</p>","course_feature":["Problem Sets","Problem Set Solutions"]}]}`,
		"/api/v1/commons/catalog":                                                                  `{"numTotal":2,"books":[{"bookID":"stats-1","title":"Introductory Statistics","author":"Shafer and Zhang","library":"stats","license":"ccbyncsa","summary":"Includes Bayesian reasoning.","links":{"online":"https://stats.libretexts.org/Bookshelves/Introductory_Statistics/Introductory_Statistics_(Shafer_and_Zhang)"}},{"bookID":"chem-9","title":"Organic Chemistry","author":"X","library":"chem","license":"ccby","summary":"Carbon.","links":{"online":"https://chem.libretexts.org/x"}}]}`,
	})
	c := client(t)
	c.Catalogues = Catalogues{OpenTextbookLibrary: srv.URL + "/opentextbooks/textbooks.json", MITLearn: srv.URL + "/api/v1/learning_resources_search/", LibreTexts: srv.URL + "/api/v1/commons/catalog"}
	got := c.Search(context.Background(), []string{"bayesian"})
	byOrigin := map[string]Candidate{}
	for _, cand := range got {
		byOrigin[cand.Origin] = cand
	}
	if len(got) != 3 {
		t.Fatalf("got %d candidates: %+v", len(got), got)
	}
	otl := byOrigin["Open Textbook Library"]
	if otl.Source.Title != "Think Bayes: Bayesian Statistics Made Simple" || otl.Source.Verdict != Adaptable || otl.Source.Fetch.Kind != "" || otl.Source.Authors[0] != "Allen Downey" {
		t.Fatalf("otl: %+v", otl)
	}
	ocw := byOrigin["MIT Learn"]
	if ocw.Source.Fetch.Kind != "ocw" || ocw.Source.Fetch.Course != "18-05-introduction-to-probability-and-statistics-spring-2022" || ocw.Source.Verdict != Adaptable || ocw.Source.Exercises == "" {
		t.Fatalf("ocw: %+v", ocw)
	}
	lt := byOrigin["LibreTexts"]
	if lt.Source.Fetch.Kind != "libretexts" || lt.Source.Title != "Introductory Statistics" || lt.Source.Licence.Name != "CC BY-NC-SA" {
		t.Fatalf("libretexts: %+v", lt)
	}
}

func TestVerdictForALicenceName(t *testing.T) {
	cases := map[string]Verdict{
		"Attribution-NonCommercial-NoDerivs": QuoteOnly, "CC BY-NC-ND 4.0": QuoteOnly, "ccbyncnd": QuoteOnly,
		"Attribution-ShareAlike": Adaptable, "CC BY 4.0": Adaptable, "ccbyncsa": Adaptable, "Public Domain": Adaptable, "CC0": Adaptable,
		"": QuoteOnly, "All Rights Reserved": Restricted,
	}
	for name, want := range cases {
		if got := VerdictFor(name); got != want {
			t.Errorf("%q: got %s, want %s", name, got, want)
		}
	}
}
