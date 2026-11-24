import { constants } from "node:fs";
import { copyFile, mkdir, readdir, stat } from "node:fs/promises";
import path from "node:path";

const distributions = {
  linux: {
    directory: "releases/desktop",
    extensions: [".AppImage", ".deb", ".rpm"],
    backend: "releases/server/lwn-server",
  },
  macos: {
    directory: "releases/desktop",
    extensions: [".dmg", ".zip"],
    backend: "releases/server/lwn-server",
  },
  windows: {
    directory: "releases/desktop/build",
    extensions: [".exe", ".exe"],
    backend: "apps/desktop/assets/lwn-server.exe",
  },
};

const [platform, architecture] = process.argv.slice(2);
if (!Object.hasOwn(distributions, platform) || !["x64", "arm64", "armv7l"].includes(architecture)) {
  throw new Error("Usage: node stage-release-assets.mjs <linux|macos|windows> <x64|arm64|armv7l>");
}

const { directory, extensions, backend } = distributions[platform];
const entries = await readdir(directory, { withFileTypes: true });
const desktopFiles = entries
  .filter((entry) => entry.isFile() && extensions.includes(path.extname(entry.name)))
  .map((entry) => entry.name);
const actualExtensions = desktopFiles.map((name) => path.extname(name)).sort();

if (actualExtensions.join(",") !== [...extensions].sort().join(",")) {
  throw new Error(`Expected ${extensions.join(", ")} in ${directory}, found ${desktopFiles.join(", ")}`);
}

const stageDirectory = "release-assets";
await mkdir(stageDirectory);
const suffix = `${platform}-${architecture}`;

async function stage(source, name) {
  if (!(await stat(source)).size) {
    throw new Error(`Empty release asset: ${source}`);
  }

  // Use only characters preserved by GitHub in uploaded asset names.
  const destination = path.join(stageDirectory, name.replace(/[^a-zA-Z0-9._-]/g, "-"));
  await copyFile(source, destination, constants.COPYFILE_EXCL);
  console.log(destination);
}

for (const name of desktopFiles) {
  await stage(path.join(directory, name), `${suffix}-${name}`);
}
await stage(backend, `lwn-server-${suffix}${path.extname(backend)}`);
