import * as child_process from "child_process";
import { promisify } from "util";
import { Readable } from "stream";
import * as path from "path";
import * as fs from "fs";
import * as process from "process";
import * as vscode from "vscode";

const execFile = promisify(child_process.execFile);

export async function mroFormat(
    fileContent: string,
    fileName: string,
    formatImports: boolean,
    mropath: string,
    token: vscode.CancellationToken,
): Promise<string> {
    const args = [`format`, `--stdin`];
    if (formatImports) {
        args.push(`--includes`);
    }
    if (fileName) {
        args.push(fileName);
    }
    const workspacePath = vscode.workspace.getWorkspaceFolder(
        vscode.Uri.file(fileName))?.uri;
    return (await executeMro(
        workspacePath, fileContent, args, mropath, token)).stdout;
}

function substituteWorkspace(filePath: string, workspacePath?: vscode.Uri) {
    return filePath.replace(
        "${workspaceFolder}",
        workspacePath?.fsPath ?? "."
    )
}

async function getDefaultMroExecutablePath(
    workspacePath?: vscode.Uri): Promise<string> {
    // Try to retrieve the executable from VS Code's settings. If it's not set,
    // just use "mro" as the default and get it from the system PATH.
    const mroConfig = vscode.workspace.getConfiguration("martian-lang");
    let mroExecutable = mroConfig.get<string>("mroExecutable");
    if (!mroExecutable || mroExecutable.length === 0) {
        return "mro";
    }
    mroExecutable = substituteWorkspace(mroExecutable, workspacePath)
    if (!path.isAbsolute(mroExecutable)) {
        try {
            await fs.promises.access(mroExecutable, fs.constants.R_OK);
        } catch {
            if (workspacePath) {
                return vscode.Uri.joinPath(
                    workspacePath, mroExecutable).fsPath;
            } else {
                mroExecutable = path.basename(mroExecutable);
                if (!mroExecutable) {
                    mroExecutable = "mro";
                }
            }
        }
    }
    return mroExecutable;
}

function processMroPath(
    mropath: string,
    workspacePath?: vscode.Uri): string {
    mropath = substituteWorkspace(mropath, workspacePath);
    let paths = mropath.split(":");
    if (workspacePath) {
        paths = paths.map(p => path.isAbsolute(p) ?
            p : vscode.Uri.joinPath(workspacePath, p).fsPath);
    }
    return paths.join(":");
}

function getMroEnv(
    mropath: string,
    workspacePath?: vscode.Uri): {
        [key: string]: string | undefined
    } {
    if (mropath === "") {
        return process.env;
    }
    const env = { ...process.env };
    mropath = processMroPath(mropath, workspacePath);
    env.MROPATH = mropath;
    return env;
}

async function executeMro(
    workspacePath: vscode.Uri | undefined,
    fileContent: string,
    args: string[],
    mropath: string,
    token: vscode.CancellationToken,
): Promise<{ stdout: string; stderr: string }> {
    const execOptions: child_process.ExecOptions = {
        env: getMroEnv(mropath, workspacePath),
        maxBuffer: Number.MAX_SAFE_INTEGER,
    };
    const exePath = await getDefaultMroExecutablePath(workspacePath);
    if (token.isCancellationRequested) {
        return { stdout: "", stderr: "" };
    }
    const procPromise = execFile(
        exePath,
        args,
        execOptions,
    );
    const proc = procPromise.child;
    token.onCancellationRequested(() => proc.kill());
    if (token.isCancellationRequested) {
        proc.kill();
        if (proc.stdin) {
            proc.stdin.end();
        }
    } else if (proc.stdin) {
        Readable.from([fileContent]).pipe(proc.stdin, { end: true });
    }
    return await procPromise;
}
