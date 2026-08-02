# md2a2ui

Go library that converts Markdown to [A2UI](https://a2ui.org) component trees using [goldmark](https://github.com/yuin/goldmark).

## Install

```bash
go get gitea.stump.rocks/stump.wtf/md2a2ui
```

## Usage

```go
package main

import (
    "encoding/json"
    "fmt"
    
    "gitea.stump.rocks/stump.wtf/md2a2ui"
)

func main() {
    md := `# Hello World

This is **bold** and *italic*.

- Item one
- Item two

| Column A | Column B |
|----------|----------|
| Cell 1   | Cell 2   |
`

    msg, err := md2a2ui.Convert(md)
    if err != nil {
        panic(err)
    }
    
    out, _ := json.MarshalIndent(msg, "", "  ")
    fmt.Println(string(out))
}
```

## Markdown → A2UI Mapping

| Markdown | A2UI Component |
|----------|---------------|
| `# H1`–`# H5` | `Text` (variant: h1–h5) |
| `###### H6` | `Text` (variant: caption) |
| Paragraph | `Text` (variant: body) |
| Bullet/numbered list | `List` |
| Code block | `Card` wrapping `Text` |
| Blockquote | `Card` wrapping `Text` |
| `---` (horizontal rule) | `Divider` |
| Image | `Image` |
| Table | `Column` of `Row` children |
| Inline bold/italic/code/links | Preserved as markdown in `Text.text` |

## License

MIT
