package main

import "fmt"

func validateSampleStatus(status string) error {
	if status == "" {
		return fmt.Errorf("status is required")
	}
	for _, value := range []string{"in_review", "released", "rejected"} {
		if status == value {
			return nil
		}
	}
	return fmt.Errorf("unsupported status %q", status)
}
