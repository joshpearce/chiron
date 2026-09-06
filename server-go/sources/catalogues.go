package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// Catalogues are the keyless search endpoints of OPEN-SOURCES.md section
// 3.1, asked when a brief names no source the index has.
type Catalogues struct {
	OpenTextbookLibrary string
	MITLearn            string
	LibreTexts          string
}

var DefaultCatalogues = Catalogues{
	OpenTextbookLibrary: "https://open.umn.edu/opentextbooks/textbooks.json",
	MITLearn:            "https://api.learn.mit.edu/api/v1/learning_resources_search/",
	LibreTexts:          "https://commons.libretexts.org/api/v1/commons/catalog",
}

// A Candidate is a source a catalogue offered for a brief. Its Source is
// provisional: a verdict read from the catalogue's licence field, and a
// fetch recipe only where a fetcher exists for the host, so it is what
// the planner sees before anything is fetched.
type Candidate struct {
	Source      Source
	Origin      string
	Description string
}

// VerdictFor reads a licence name the way section 4 of OPEN-SOURCES.md
// does: derivatives allowed means adaptable, no-derivatives means
// quotation only, and reserved or absent means nothing is fetched.
func VerdictFor(licence string) Verdict {
	l := strings.ToLower(licence)
	compact := strings.NewReplacer(" ", "", "-", "", "_", "").Replace(l)
	switch {
	case l == "":
		return QuoteOnly
	case strings.Contains(l, "all rights reserved") || compact == "arr" || strings.Contains(l, "copyright"):
		return Restricted
	case strings.Contains(l, "nd") && (strings.Contains(compact, "ccby") || strings.Contains(l, "attribution")), strings.Contains(l, "noderiv"):
		return QuoteOnly
	case strings.Contains(l, "attribution") || strings.Contains(compact, "ccby") || strings.Contains(l, "cc0") ||
		strings.Contains(l, "public domain") || strings.Contains(compact, "publicdomain") || strings.Contains(l, "gfdl") || strings.Contains(l, "gnu"):
		return Adaptable
	}
	return QuoteOnly
}

// licenceName spells out the codes the catalogues use.
func licenceName(code string) string {
	names := map[string]string{
		"ccby": "CC BY", "ccbysa": "CC BY-SA", "ccbync": "CC BY-NC", "ccbyncsa": "CC BY-NC-SA",
		"ccbynd": "CC BY-ND", "ccbyncnd": "CC BY-NC-ND", "publicdomain": "Public Domain", "gnu": "GNU GPL",
		"gnufdl": "GNU FDL", "arr": "All Rights Reserved", "notset": "",
	}
	if n, ok := names[strings.ToLower(strings.TrimSpace(code))]; ok {
		return n
	}
	return strings.NewReplacer("Attribution", "CC BY", "-NonCommercial", "-NC", "-ShareAlike", "-SA", "-NoDerivs", "-ND", "-NoDerivatives", "-ND").Replace(code)
}

var ocwCourseURL = regexp.MustCompile(`ocw\.mit\.edu/courses/([a-z0-9.-]+)`)

// Search asks every catalogue for every term and returns the candidates,
// one per title, in the order found. A catalogue that fails is skipped:
// the index is the finder's floor, the catalogues its reach.
func (c *Client) Search(ctx context.Context, terms []string) []Candidate {
	var out []Candidate
	seen := map[string]bool{}
	add := func(cand Candidate) {
		key := strings.ToLower(strings.TrimSpace(cand.Source.Title))
		if key == "" || seen[key] {
			return
		}
		seen[key] = true
		if cand.Source.ID == "" {
			cand.Source.ID = slugID(cand.Origin, cand.Source.Title)
		}
		out = append(out, cand)
	}
	for _, term := range terms {
		for _, cand := range c.openTextbookLibrary(ctx, term) {
			add(cand)
		}
		for _, cand := range c.mitLearn(ctx, term) {
			add(cand)
		}
	}
	for _, cand := range c.libreTexts(ctx, terms) {
		add(cand)
	}
	return out
}

var idBad = regexp.MustCompile(`[^a-z0-9]+`)

func slugID(origin, title string) string {
	prefix := map[string]string{"Open Textbook Library": "otl", "MIT Learn": "ocw", "LibreTexts": "libretexts"}[origin]
	if prefix == "" {
		prefix = "found"
	}
	t := strings.Trim(idBad.ReplaceAllString(strings.ToLower(title), "-"), "-")
	if len(t) > 50 {
		t = t[:50]
	}
	return prefix + "-" + t
}

func (c *Client) openTextbookLibrary(ctx context.Context, term string) []Candidate {
	if c.Catalogues.OpenTextbookLibrary == "" {
		return nil
	}
	data, err := c.get(ctx, c.Catalogues.OpenTextbookLibrary+"?q="+url.QueryEscape(term))
	if err != nil {
		return nil
	}
	var reply struct {
		Data []struct {
			Title        string `json:"title"`
			Licence      string `json:"license"`
			Description  string `json:"description"`
			URL          string `json:"url"`
			Contributors []struct {
				Contribution string `json:"contribution"`
				First        string `json:"first_name"`
				Last         string `json:"last_name"`
			} `json:"contributors"`
			Formats []struct {
				Type string `json:"type"`
				URL  string `json:"url"`
			} `json:"formats"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &reply); err != nil {
		return nil
	}
	var out []Candidate
	for _, r := range reply.Data {
		s := Source{Title: r.Title, Licence: Licence{Name: licenceName(r.Licence), URL: r.URL}, Verdict: VerdictFor(r.Licence), Format: "Open Textbook Library record"}
		for _, p := range r.Contributors {
			if p.Contribution == "" || strings.EqualFold(p.Contribution, "Author") {
				s.Authors = append(s.Authors, strings.TrimSpace(p.First+" "+p.Last))
			}
		}
		for _, f := range r.Formats {
			if m := ocwCourseURL.FindStringSubmatch(f.URL); m != nil {
				s.Fetch = Recipe{Kind: "ocw", Course: m[1]}
			} else if strings.Contains(f.URL, ".libretexts.org/") {
				s.Fetch = Recipe{Kind: "libretexts", URL: f.URL}
			}
		}
		out = append(out, Candidate{Source: s, Origin: "Open Textbook Library", Description: strings.TrimSpace(r.Description)})
	}
	return out
}

func (c *Client) mitLearn(ctx context.Context, term string) []Candidate {
	if c.Catalogues.MITLearn == "" {
		return nil
	}
	data, err := c.get(ctx, c.Catalogues.MITLearn+"?q="+url.QueryEscape(term)+"&platform=ocw&resource_type=course&limit=10")
	if err != nil {
		return nil
	}
	var reply struct {
		Results []struct {
			ReadableID  string   `json:"readable_id"`
			Title       string   `json:"title"`
			URL         string   `json:"url"`
			Description string   `json:"description"`
			Features    []string `json:"course_feature"`
		} `json:"results"`
	}
	if err := json.Unmarshal(data, &reply); err != nil {
		return nil
	}
	var out []Candidate
	for _, r := range reply.Results {
		m := ocwCourseURL.FindStringSubmatch(r.URL)
		if m == nil {
			continue
		}
		number := strings.SplitN(r.ReadableID, "+", 2)[0]
		s := Source{ID: "ocw-" + strings.Trim(idBad.ReplaceAllString(strings.ToLower(number), "-"), "-"),
			Title: fmt.Sprintf("MIT %s %s", number, r.Title), Authors: []string{"MIT OpenCourseWare"},
			Licence: Licence{Name: "CC BY-NC-SA 4.0", URL: "https://ocw.mit.edu/pages/privacy-and-terms-of-use/"}, Verdict: Adaptable,
			Format: "OCW course pages and PDFs", Fetch: Recipe{Kind: "ocw", Course: m[1]}}
		var has []string
		for _, f := range r.Features {
			if strings.Contains(f, "Problem") || strings.Contains(f, "Exam") || strings.Contains(f, "Assignment") {
				has = append(has, f)
			}
		}
		if len(has) > 0 {
			s.Exercises = strings.Join(has, ", ") + " as PDFs"
		}
		out = append(out, Candidate{Source: s, Origin: "MIT Learn", Description: stripTags(r.Description)})
	}
	return out
}

var tagRe = regexp.MustCompile(`<[^>]+>`)

func stripTags(s string) string {
	return strings.TrimSpace(tagRe.ReplaceAllString(s, ""))
}

// libreTexts filters the whole catalogue (one cached call) on the terms.
func (c *Client) libreTexts(ctx context.Context, terms []string) []Candidate {
	if c.Catalogues.LibreTexts == "" || len(terms) == 0 {
		return nil
	}
	data, err := c.get(ctx, c.Catalogues.LibreTexts)
	if err != nil {
		return nil
	}
	var reply struct {
		Books []struct {
			ID      string `json:"bookID"`
			Title   string `json:"title"`
			Author  string `json:"author"`
			Library string `json:"library"`
			Licence string `json:"license"`
			Summary string `json:"summary"`
			Links   struct {
				Online string `json:"online"`
			} `json:"links"`
		} `json:"books"`
	}
	if err := json.Unmarshal(data, &reply); err != nil {
		return nil
	}
	var out []Candidate
	for _, b := range reply.Books {
		hay := strings.ToLower(b.Title + " " + b.Summary)
		hit := false
		for _, t := range terms {
			if strings.Contains(hay, strings.ToLower(t)) {
				hit = true
			}
		}
		if !hit || b.Links.Online == "" || b.Licence == "" {
			continue
		}
		s := Source{ID: "libretexts-" + strings.Trim(idBad.ReplaceAllString(strings.ToLower(b.ID), "-"), "-"),
			Title: b.Title, Licence: Licence{Name: licenceName(b.Licence), URL: b.Links.Online}, Verdict: VerdictFor(licenceName(b.Licence)),
			Format: "LibreTexts (" + b.Library + ")", Fetch: Recipe{Kind: "libretexts", URL: b.Links.Online}}
		if b.Author != "" {
			s.Authors = []string{b.Author}
		}
		out = append(out, Candidate{Source: s, Origin: "LibreTexts", Description: strings.TrimSpace(b.Summary)})
		if len(out) >= 10 {
			break
		}
	}
	return out
}
