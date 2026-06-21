const source = "web/elm/src/Main.elm";
const output = "web/static/app.js";
const elmCompiler = Deno.env.get("ELM_BIN");

if (elmCompiler === undefined || elmCompiler.length === 0) {
  console.error("ELM_BIN must point to the Elm 0.19.1 compiler.");
  Deno.exit(1);
}

const command = new Deno.Command(elmCompiler, {
  args: ["make", source, "--output", output],
  stdout: "inherit",
  stderr: "inherit",
});

const status = await command.output();
if (!status.success) {
  Deno.exit(status.code);
}
