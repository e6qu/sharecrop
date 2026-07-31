// Copy the compiled app stylesheet and the self-hosted fonts into the demo
// bundle. The stylesheet references fonts as /static/fonts/... (the app shell
// serves it from /static/app.css); the demo serves the same stylesheet as
// ./app.css under an arbitrary base path (locally /, on Pages /<repo>/demo/),
// so the copy rewrites those URLs to be relative to the stylesheet and the
// font files are copied alongside it under site/demo/fonts/.
const css = await Deno.readTextFile("web/static/app.css");
await Deno.writeTextFile(
  "site/demo/app.css",
  css.replaceAll("/static/fonts/", "fonts/"),
);

await Deno.mkdir("site/demo/fonts", { recursive: true });
for await (const entry of Deno.readDir("web/static/fonts")) {
  if (entry.isFile) {
    await Deno.copyFile(
      `web/static/fonts/${entry.name}`,
      `site/demo/fonts/${entry.name}`,
    );
  }
}
