import fs from "node:fs";

fs.rmSync("dist", { force: true, recursive: true });
