package services

import (
	"reflect"
	"testing"
)

func TestAppendUnique(t *testing.T) {
	got := appendUnique(nil, "a")
	if !reflect.DeepEqual(got, []string{"a"}) {
		t.Fatalf("nil + a = %v, want [a]", got)
	}
	got = appendUnique([]string{"a"}, "b")
	if !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("got %v, want [a b]", got)
	}
	got = appendUnique([]string{"a"}, "A") // case-insensitive dedupe
	if !reflect.DeepEqual(got, []string{"a"}) {
		t.Fatalf("case-insensitive dedupe failed, got %v", got)
	}
	got = appendUnique([]string{"foo", "bar"}, "foo")
	if !reflect.DeepEqual(got, []string{"foo", "bar"}) {
		t.Fatalf("exact dedupe failed, got %v", got)
	}
}

func TestCleanText(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"hello", "hello"},
		{"hello   world", "hello world"},
		{"  leading and trailing  ", "leading and trailing"},
		{"tabs\tand\nnewlines", "tabs and newlines"},
		{"\t\t\t", ""},
	}
	for _, c := range cases {
		if got := cleanText(c.in); got != c.want {
			t.Errorf("cleanText(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestInferIndustry(t *testing.T) {
	cases := []struct {
		desc string
		want string
	}{
		{"We build cloud SaaS software", "Technology"},
		{"Industrial manufacturing of metal goods", "Manufacturing"},
		{"Italian assicurazione provider", "Finance"},
		{"Sanità e farmaci", "Healthcare"},
		{"Online retail shop", "E-commerce"},
		{"Strategic consulting and advisory", "Consulting"},
		{"Edil costruzioni civili", "Construction"},
		{"Ristorazione e alimentare", "Food & Beverage"},
		{"Logistic transport services", "Logistics"},
		{"Education and training programs", "Education"},
		{"Generic widget company", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := inferIndustry(c.desc); got != c.want {
			t.Errorf("inferIndustry(%q) = %q, want %q", c.desc, got, c.want)
		}
	}
}
