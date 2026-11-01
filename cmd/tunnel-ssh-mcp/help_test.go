// Created by Yanjunhui
package main

import "testing"

func TestSubcommandHelpSucceedsWithoutConfiguration(t *testing.T) {
	for _, command := range []string{"run", "serve", "doctor", "check", "install-tunnel", "init"} {
		t.Run(command, func(t *testing.T) {
			if err := run(t.Context(), []string{command, "--help"}); err != nil {
				t.Fatalf("help must succeed without loading configuration or starting services: %v", err)
			}
		})
	}
}
