
Article markdown files will be placed in the blog/articles/ directory.
Each article filename follows the format YYYY-MM-DD-file-name.md
Parse the date from the filename for later use as the article publish date.
If the filename does not match the expected format, skip the file and emit a warning log message describing which files were skipped and why.
Parse the title as the first '#' header from the article markdown file for later use when building the article index.

Generate as associated article html file and place it in the public/articles/ directory. The generated html file should have the same name as the original markdown file except the file extension 'md' is replaced with 'html'.

In the generated index.html file, insert an 'Articles' section after the index.md content. 
The articles section should list each article similar to the following:

- [2025-06-12] Article Title 2
- [2025-05-01] Article Title 1

Notice that the list is sorted from newest to oldest articles.

Each article in the list should have a link to the relative path of the html generated article.

