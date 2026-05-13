package models

import "testing"

func TestGrade(t *testing.T) {
	cases := []struct {
		score int
		want  string
	}{
		// A boundary
		{100, "A"},
		{90, "A"},
		// B boundary
		{89, "B"},
		{75, "B"},
		// C boundary
		{74, "C"},
		{60, "C"},
		// D boundary
		{59, "D"},
		{40, "D"},
		// F
		{39, "F"},
		{0, "F"},
		// Out of band — Grade has no clamp; F covers both
		{-1, "F"},
		{1000, "A"},
	}
	for _, c := range cases {
		if got := Grade(c.score); got != c.want {
			t.Errorf("Grade(%d) = %q, want %q", c.score, got, c.want)
		}
	}
}

func TestValidSkillsMembership(t *testing.T) {
	// Spot-check a few canonical entries; this also keeps the global map referenced.
	for _, s := range []SkillName{SkillResearch, SkillQualify, SkillContacts, SkillOutreach, SkillFollowup, SkillPrep, SkillProposal, SkillObjections, SkillICP, SkillCompetitors} {
		if !ValidSkills[s] {
			t.Errorf("ValidSkills missing canonical skill %q", s)
		}
	}
	if ValidSkills[SkillName("bogus")] {
		t.Errorf("ValidSkills should not contain unknown skill 'bogus'")
	}
}
