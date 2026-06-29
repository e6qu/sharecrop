const source = "web/elm/src/Main.elm";
const output = "web/static/app.js";
const demoOutput = "site/demo/elm.js";
const elmCompiler = Deno.env.get("ELM_BIN");

if (elmCompiler === undefined || elmCompiler.length === 0) {
  console.error("ELM_BIN must point to the Elm 0.19.1 compiler.");
  Deno.exit(1);
}

await rejectRecursiveNpmShim(elmCompiler);

const command = new Deno.Command(elmCompiler, {
  args: ["make", source, "--output", output],
  stdout: "inherit",
  stderr: "inherit",
});

const status = await command.output();
if (!status.success) {
  Deno.exit(status.code);
}

await Deno.copyFile(output, demoOutput);

async function rejectRecursiveNpmShim(path: string): Promise<void> {
  let contents = "";
  try {
    contents = await Deno.readTextFile(path);
  } catch (_error) {
    return;
  }

  if (
    contents.includes("var child_process = require('child_process')") &&
    contents.includes("path.resolve(__dirname, 'elm')") &&
    contents.includes("shell: true")
  ) {
    console.error(
      [
        "ELM_BIN points to the npm Elm wrapper, not the native Elm compiler.",
        "In this install that wrapper can recursively spawn itself and hang.",
        "Set ELM_BIN to a native Elm 0.19.1 binary, for example /opt/homebrew/bin/elm.",
      ].join("\n"),
    );
    Deno.exit(1);
  }
}
