# skim

A terminal-based speed reader that displays text using the Rapid Serial Visual Presentation (RSVP) technique with Optimal Recognition Point (ORP) highlighting.

![skim](public/demo.png)

## Installation

```bash
go install github.com/varunrandery/skim@latest
```

Or build from source:

```bash
go build -o skim
```

## Usage

```bash
skim [options] [file|url]
```

```bash
skim document.txt
skim -wpm 400 article.md
skim http://httpbin.org/html
cat book.md | skim
llm 'Explain what stdin is' | skim
skim # Opens file picker
```

## License

MIT
