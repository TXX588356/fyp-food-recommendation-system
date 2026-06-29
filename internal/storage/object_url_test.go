package storage

import "testing"

func TestObjectURLResolverEscapesEachObjectKeySegment(t *testing.T) {
	resolver := NewObjectURLResolver("http://localhost:9000/", "images")

	got := resolver.Resolve("catalog meals/meal one/image #1.jpg")
	want := "http://localhost:9000/images/catalog%20meals/meal%20one/image%20%231.jpg"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestObjectURLResolverReturnsEmptyForEmptyKey(t *testing.T) {
	resolver := NewObjectURLResolver("http://localhost:9000", "images")
	if got := resolver.Resolve("  "); got != "" {
		t.Fatalf("expected empty URL, got %q", got)
	}
}
