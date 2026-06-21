Deno.test("deno test harness runs", () => {
  if (Deno.build.os.length === 0) {
    throw new Error("missing deno build metadata");
  }
});
