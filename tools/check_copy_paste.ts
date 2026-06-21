type Fingerprint = {
  path: string;
  line: number;
};

type Duplicate = {
  first: Fingerprint;
  second: Fingerprint;
};

const roots: string[] = ["cmd", "internal", "tests", "tools", "web/elm/src"];
const generatedSuffixes: string[] = ["web/static/app.js", "web/static/app.css"];
const windowSize = 12;
const minimumLineLength = 8;

async function collectFiles(directory: string, files: string[]): Promise<void> {
  for await (const entry of Deno.readDir(directory)) {
    const path = `${directory}/${entry.name}`;
    if (entry.isDirectory) {
      await collectFiles(path, files);
      continue;
    }

    if (entry.isFile) {
      files.push(path);
    }
  }
}

function shouldSkip(path: string): boolean {
  for (const suffix of generatedSuffixes) {
    if (path.endsWith(suffix)) {
      return true;
    }
  }

  return false;
}

function normalize(source: string): string[] {
  return source
    .split("\n")
    .map((line: string): string => line.trim())
    .filter((line: string): boolean => line.length >= minimumLineLength)
    .filter((line: string): boolean => !line.startsWith("//"))
    .filter((line: string): boolean => !line.startsWith("#"));
}

const files: string[] = [];
for (const root of roots) {
  await collectFiles(root, files);
}

const seen = new Map<string, Fingerprint>();
const duplicates: Duplicate[] = [];

for (const path of files) {
  if (shouldSkip(path)) {
    continue;
  }

  const lines = normalize(await Deno.readTextFile(path));
  for (let index = 0; index + windowSize <= lines.length; index += 1) {
    const key = lines.slice(index, index + windowSize).join("\n");
    const found = seen.get(key);
    const current = { path, line: index + 1 };

    if (found !== undefined) {
      duplicates.push({ first: found, second: current });
      continue;
    }

    seen.set(key, current);
  }
}

if (duplicates.length > 0) {
  for (const duplicate of duplicates) {
    console.error(
      `copy-paste block: ${duplicate.first.path}:${duplicate.first.line} and ${duplicate.second.path}:${duplicate.second.line}`,
    );
  }

  Deno.exit(1);
}
