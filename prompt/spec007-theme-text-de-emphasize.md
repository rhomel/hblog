Refactor and extract the theme struct and loadTheme function into a separate Go
file in the same package: `theme.go`.

---

Update theme.go to parse a new property `text-de-emphasize` which includes a css color value:

```
- text-de-emphasize: #676767
```

Update main.go's index file generation to use this new property as the color of
the date text.
Update the date format to exclude the '[]' characters. Example:

```
2025-06-15 blog title
```

---

Update theme.go to parse a new property `article-line-height` which includes a css line-height value:

```
- article-line-height: 1.5
```

---

Update main.go to use the theme article-line-height to apply as the line height
for the generated HTML article paragraph text.
