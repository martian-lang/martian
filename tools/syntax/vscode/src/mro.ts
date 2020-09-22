import * as child_process from "child_process";
import * as process from "process";
import * as vscode from "vscode";

export async function mroFormat(
    fileContent: string,
    fileName: string,
    formatImports: boolean,
    mropath: string,
): Promise<string> {
    const args = [`format`, `--stdin`];
    if (formatImports) {
        args.push(`--includes`);
    }
    if (fileName) {
        args.push(fileName);
    }
    return (await executeMro(fileContent, args, mropath)).stdout;
}

function getDefaultMroExecutablePath(): string {
    // Try to retrieve the executable from VS Code's settings. If it's not set,
    // just use "mro" as the default and get it from the system PATH.
    const mroConfig = vscode.workspace.getConfiguration("martian-lang");
    const mroExecutable = mroConfig.get<string>("mroExecutable");
    if (mroExecutable.length === 0) {
        return "mro";
    }
    return mroExecutable;
}

function getMroEnv(mropath: string): any {
    if (mropath === "") {
        return process.env;
    }
    const env = { ...process.env };
    env.MROPATH = mropath;
    return env;
}

function executeMro(
    fileContent: string,
    args: string[],
    mropath: string,
): Promise<{ stdout: string; stderr: string }> {
    return new Promise((resolve, reject) => {
        const execOptions = {
            env: getMroEnv(mropath),
            maxBuffer: Number.MAX_SAFE_INTEGER,
        };
        const proc = child_process.execFile(
            getDefaultMroExecutablePath(),
            args,
            execOptions,
            (error: Error, stdout: string, stderr: string) => {
                if (!error) {
                    resolve({ stdout, stderr });
                } else {
                    reject(error);
                }
            },
        );
        // Write the file being formatted to stdin and close the stream so
        // that the mro process continues.
        proc.stdin.write(fileContent);
        proc.stdin.end();
    });
}
