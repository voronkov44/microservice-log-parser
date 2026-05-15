package main

import "testing"

func TestMaskDSN(t *testing.T) {
	got := maskDSN("postgres://postgres:secret@postgres:5432/postgres?sslmode=disable")
	want := "postgres://postgres:xxxxx@postgres:5432/postgres?sslmode=disable"

	if got != want {
		t.Fatalf("maskDSN() = %q, want %q", got, want)
	}
}

func TestMaskDSNWithoutPassword(t *testing.T) {
	got := maskDSN("postgres://postgres@postgres:5432/postgres?sslmode=disable")
	want := "postgres://postgres@postgres:5432/postgres?sslmode=disable"

	if got != want {
		t.Fatalf("maskDSN() = %q, want %q", got, want)
	}
}
