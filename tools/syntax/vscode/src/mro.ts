import * as child_process from "child_process";
import * as path from "path";
import * as fs from "fs";
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
    const workspacePath = vscode.workspace.getWorkspaceFolder(
        vscode.Uri.file(fileName)).uri;
    return (await executeMro(workspacePath, fileContent, args, mropath)).stdout;
}

async function getDefaultMroExecutablePath(
    workspacePath: vscode.Uri): Promise<string> {
    // Try to retrieve the executable from VS Code's settings. If it's not set,
    // just use "mro" as the default and get it from the system PATH.
    const mroConfig = vscode.workspace.getConfiguration("martian-lang");
    let mroExecutable = mroConfig.get<string>("mroExecutable");
    if (mroExecutable.length === 0) {
        return "mro";
    }
    mroExecutable = mroExecutable.replace(
        "${workspaceFolder}",
        workspacePath.fsPath
    )
    if (!path.isAbsolute(mroExecutable)) {
        try {
            await fs.promises.access(mroExecutable, fs.constants.R_OK);
        } catch {
            mroExecutable = vscode.Uri.joinPath(
                workspacePath, mroExecutable).fsPath;
        }
    }
    return mroExecutable;
}

function getMroEnv(mropath: string, workspacePath: vscode.Uri): any {
    if (mropath === "") {
        return process.env;
    }
    const env = { ...process.env };
    mropath = mropath.replace(
        "${workspaceFolder}",
        workspacePath.fsPath
    );
    if (!path.isAbsolute(mropath)) {
        mropath = vscode.Uri.joinPath(workspacePath, mropath).fsPath;
    }
    env.MROPATH = mropath;
    return env;
}

function executeMro(
    workspacePath: vscode.Uri,
    fileContent: string,
    args: string[],
    mropath: string,
): Promise<{ stdout: string; stderr: string }> {
    return new Promise(async (resolve, reject) => {
        const execOptions = {
            env: getMroEnv(mropath, workspacePath),
            maxBuffer: Number.MAX_SAFE_INTEGER,
        };
        const proc = child_process.execFile(
            await getDefaultMroExecutablePath(workspacePath),
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
