package filter_test

import (
	"testing"

	"github.com/FikranAkbar/go-backend-job-scrapper/internal/filter"
	"github.com/FikranAkbar/go-backend-job-scrapper/internal/store"
)

func TestIsRelevant(t *testing.T) {
	tests := []struct {
		name string
		job  store.Job
		want bool
	}{
		// --- include keyword passes ---
		{"golang title passes", store.Job{Title: "Golang Backend Engineer", Location: "Remote"}, true},
		{"go backend title passes", store.Job{Title: "Go Backend Developer", Location: "Remote"}, true},
		{"grpc tag passes", store.Job{Title: "Backend Engineer", Tags: []string{"grpc", "go"}, Location: "Remote"}, true},
		{"microservices tag passes", store.Job{Title: "Software Engineer", Tags: []string{"microservices", "golang"}, Location: "Remote"}, true},
		{"platform engineer passes", store.Job{Title: "Platform Engineer", Location: "Remote"}, true},
		{"distributed systems passes", store.Job{Title: "Distributed Systems Engineer", Location: "Remote"}, true},
		{"api engineer passes", store.Job{Title: "API Engineer", Location: "Worldwide"}, true},
		{"rest api tag passes", store.Job{Title: "Backend Engineer", Tags: []string{"rest api"}, Location: "Remote"}, true},
		{"software engineer backend passes", store.Job{Title: "Software Engineer Backend", Location: "Remote"}, true},

		// --- exclude keyword rejects ---
		{"frontend in title rejected", store.Job{Title: "React Frontend Developer", Location: "Remote"}, false},
		{"mobile in title rejected", store.Job{Title: "Mobile iOS Developer", Location: "Remote"}, false},
		{"data scientist rejected", store.Job{Title: "Data Scientist", Tags: []string{"python", "ml"}}, false},
		{"angular tag rejected", store.Job{Title: "Software Engineer", Tags: []string{"angular", "golang"}, Location: "Remote"}, false},
		{"flutter in title rejected", store.Job{Title: "Flutter Developer", Location: "Remote"}, false},
		{"machine learning rejected", store.Job{Title: "Machine Learning Engineer", Location: "Remote"}, false},
		{"qa engineer rejected", store.Job{Title: "QA Engineer", Tags: []string{"golang"}, Location: "Remote"}, false},
		{"ux designer rejected", store.Job{Title: "UX Designer", Tags: []string{"golang"}, Location: "Remote"}, false},
		{"devops only rejected", store.Job{Title: "DevOps Only Engineer", Tags: []string{"golang"}, Location: "Remote"}, false},

		// --- empty / no keywords ---
		{"empty title rejected", store.Job{Title: ""}, false},
		{"no include keyword rejected", store.Job{Title: "Systems Administrator", Location: "Remote"}, false},

		// --- location rules ---
		{"remote location passes", store.Job{Title: "Backend Engineer", Location: "Remote Worldwide"}, true},
		{"indonesia location passes", store.Job{Title: "Backend Engineer", Location: "Indonesia"}, true},
		{"japan location passes", store.Job{Title: "Backend Engineer", Location: "Japan"}, true},
		{"singapore location passes", store.Job{Title: "Backend Engineer", Location: "Singapore"}, true},
		{"united states location passes", store.Job{Title: "Backend Engineer", Location: "United States"}, true},
		{"asia location passes", store.Job{Title: "Backend Engineer", Location: "Asia"}, true},
		{"empty location passes (assumed remote)", store.Job{Title: "Golang Backend", Location: ""}, true},
		{"onsite jakarta passes", store.Job{Title: "Golang Backend Engineer", Location: "Onsite Jakarta"}, true},
		{"onsite non-jakarta rejected", store.Job{Title: "Golang Backend Engineer", Location: "Onsite New York"}, false},
		{"unknown location rejected", store.Job{Title: "Backend Engineer", Location: "Paris"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filter.IsRelevant(tt.job)
			if got != tt.want {
				t.Errorf("IsRelevant(%+v) = %v, want %v", tt.job, got, tt.want)
			}
		})
	}
}

func TestApply(t *testing.T) {
	jobs := []store.Job{
		{Title: "Golang Backend Engineer", Location: "Remote"},
		{Title: "React Frontend Developer", Location: "Remote"},
		{Title: "Platform Engineer", Location: "Singapore"},
		{Title: "Data Scientist"},
		{Title: "Go Backend Developer", Location: "Indonesia"},
	}

	got := filter.Apply(jobs)

	// Expect 3 jobs to pass: Golang Backend, Platform Engineer, Go Backend Developer
	if len(got) != 3 {
		t.Errorf("Apply() returned %d jobs, want 3", len(got))
	}

	for _, j := range got {
		if !filter.IsRelevant(j) {
			t.Errorf("Apply() included non-relevant job: %+v", j)
		}
	}
}

func TestApplyEmpty(t *testing.T) {
	got := filter.Apply(nil)
	if len(got) != 0 {
		t.Errorf("Apply(nil) returned %d jobs, want 0", len(got))
	}
}

