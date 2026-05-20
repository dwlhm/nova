package format

import "testing"

func TestNovaNormalizesIndentationAndTrailingWhitespace(t *testing.T) {
	input := `<template target <- web>   
<surface>
<text value <- "web" /|    
/|
/|
`

	got := Nova(input)
	want := `<template target <- web>
  <surface>
    <text value <- "web" /|
  /|
/|
`
	if got != want {
		t.Fatalf("formatted source =\n%s\nwant\n%s", got, want)
	}
}

func TestNovaKeepsStateTransitionBlocksReadable(t *testing.T) {
	input := `<contract state Counter>
count: number <- 0 {
@increment -> count + 1;
};
/|`

	got := Nova(input)
	want := `<contract state Counter>
  count: number <- 0 {
    @increment -> count + 1;
  };
/|
`
	if got != want {
		t.Fatalf("formatted source =\n%s\nwant\n%s", got, want)
	}
}

func TestNovaPreservesSingleBlankLineBetweenTopLevelBlocks(t *testing.T) {
	input := `<contract state Counter>
  count: number <- 0;
/|

<template target <- web>
  <text value <- "web" /|
/|
`

	got := Nova(input)
	if got != input {
		t.Fatalf("formatted source =\n%s\nwant\n%s", got, input)
	}
}
