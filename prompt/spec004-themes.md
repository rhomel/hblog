
Themes or configurations that will be used to apply some general style to the
rendered HTML will be located in the blog/themes/ directory as markdown files.

For now the supported theme properties are:

- font-family: the standard font family to use in HTML/CSS
- font-color: the standard font color
- background-color: the standard page background color
- max-content-width: the maximum width for content in the generated HTML (index included)

These properties will be written in the theme's markdown file under the heading Properties as a list. Other markdown content in the theme file may be ignored.

Here is an example:

```
   # Example theme

   A basic monochrome theme.

   # Properties

   - font-family: sans-serif
   - font-color: black
   - background-color: #efefef
   - max-content-width: 1000px
```

Update the code to support themes.
For now default to the theme 'blog/themes/default.md'.

