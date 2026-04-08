---
title: "Basic Document"
subtitle: "A simple pdfify example"
author: "pdfify"
date: "2026-04-08"
toc-level: 2
numbersections: true
numberfrom: 2
---

# Basic Document

This is a basic document demonstrating core pdfify features.

## Text Formatting

This paragraph shows **bold text**, *italic text*, ***bold italic***, and `inline code`. You can also use ~~strikethrough~~ text.

## Lists

### Unordered

- First item
- Second item with a longer description that wraps across multiple lines to test line wrapping behavior
  - Nested item A
  - Nested item B
- Third item

### Ordered

1. Step one
2. Step two
3. Step three

## Code Block

```python
def hello(name: str) -> str:
    """Say hello to someone."""
    return f"Hello, {name}! Welcome to pdfify."

if __name__ == "__main__":
    print(hello("World"))
```

## Blockquote

> This is a blockquote. It should appear with a blue left border and
> light background, providing visual distinction from normal text.

## Table

| Language | Typing | Speed | Use Case |
|----------|--------|-------|----------|
| Python | Dynamic | Medium | Data science, scripting |
| Go | Static | Fast | Systems, CLIs, servers |
| Rust | Static | Fast | Systems, safety-critical |
| TypeScript | Static | Medium | Web applications |

## Links

Visit [pdfify on GitHub](https://github.com/jclement/pdfify) for the source code.

---

*End of basic example.*
