import { spawn, type ChildProcessWithoutNullStreams } from "node:child_process";
import { once } from "node:events";

export type ProjectApiServer = {
  process: ChildProcessWithoutNullStreams;
  url: string;
};

export async function startProjectApiServer(): Promise<ProjectApiServer> {
  const process = spawn(
    "go",
    ["-C", "../core-api", "run", "./internal/testserver/project"],
    { stdio: "pipe" },
  );
  let stderr = "";
  process.stderr.setEncoding("utf8");
  process.stderr.on("data", (chunk: string) => {
    stderr += chunk;
  });

  const url = await new Promise<string>((resolve, reject) => {
    let stdout = "";
    const timeout = setTimeout(() => {
      process.kill();
      reject(new Error(`project test server timed out\n${stderr}`));
    }, 45_000);

    process.stdout.setEncoding("utf8");
    process.stdout.on("data", (chunk: string) => {
      stdout += chunk;
      const newline = stdout.indexOf("\n");
      if (newline < 0) return;

      clearTimeout(timeout);
      resolve(stdout.slice(0, newline).trim());
    });
    process.once("error", (error) => {
      clearTimeout(timeout);
      reject(error);
    });
    process.once("exit", (code) => {
      clearTimeout(timeout);
      reject(
        new Error(`project test server exited with code ${code}\n${stderr}`),
      );
    });
  });

  return { process, url };
}

export async function stopProjectApiServer(
  server: ProjectApiServer,
): Promise<void> {
  if (server.process.exitCode !== null) return;

  server.process.stdin.end();
  await Promise.race([
    once(server.process, "exit"),
    new Promise<void>((resolve) => {
      setTimeout(() => {
        server.process.kill();
        resolve();
      }, 5_000);
    }),
  ]);
}
