async function goRoot(): Promise<string> {
  const command = new Deno.Command("go", { args: ["env", "GOROOT"] });
  const output = await command.output();
  if (!output.success) {
    throw new Error("go env GOROOT failed");
  }
  const root = new TextDecoder().decode(output.stdout).trim();
  if (!root) {
    throw new Error("go env GOROOT returned an empty path");
  }
  return root;
}

async function copyWasmExec(root: string): Promise<void> {
  const candidates = [
    `${root}/misc/wasm/wasm_exec.js`,
    `${root}/lib/wasm/wasm_exec.js`,
  ];
  for (const candidate of candidates) {
    try {
      await Deno.copyFile(candidate, "site/demo/wasm_exec.js");
      return;
    } catch (errorValue) {
      if (!(errorValue instanceof Deno.errors.NotFound)) {
        throw errorValue;
      }
    }
  }
  throw new Error(`wasm_exec.js was not found under ${root}`);
}

async function buildWasm(): Promise<void> {
  const command = new Deno.Command("go", {
    args: [
      "build",
      "-o",
      "site/demo/sharecrop-wasm-backend.wasm",
      "./cmd/sharecrop-wasm",
    ],
    env: {
      GOOS: "js",
      GOARCH: "wasm",
    },
  });
  const output = await command.output();
  if (!output.success) {
    const stderr = new TextDecoder().decode(output.stderr);
    throw new Error(`go wasm build failed: ${stderr}`);
  }
}

async function generateSeedSnapshot(): Promise<void> {
  const command = new Deno.Command("go", {
    args: ["run", "./cmd/gen-seed-snapshot"],
  });
  const output = await command.output();
  if (!output.success) {
    const stderr = new TextDecoder().decode(output.stderr);
    throw new Error(`seed snapshot generation failed: ${stderr}`);
  }
}

await buildWasm();
await copyWasmExec(await goRoot());
await generateSeedSnapshot();
console.log("built site/demo/sharecrop-wasm-backend.wasm");
