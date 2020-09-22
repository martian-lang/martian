import * as vscode from "vscode";
import { mroFormat } from "./mro";

/**
 * Provides document formatting functionality for Bazel files by invoking
 * buildifier.
 */
export class MroFormatProvider
    implements vscode.DocumentFormattingEditProvider {
    public async provideDocumentFormattingEdits(
        document: vscode.TextDocument,
        options: vscode.FormattingOptions,
        token: vscode.CancellationToken,
    ): Promise<vscode.TextEdit[]> {
        const mroConfig = vscode.workspace.getConfiguration("martian-lang");
        const formatImports = mroConfig.get<boolean>("mroFormatImports");
        const mropath = mroConfig.get<string>("mropath");

        const fileContent = document.getText();
        try {
            const formattedContent = await mroFormat(
                fileContent,
                document.fileName,
                formatImports,
                mropath,
            );
            if (formattedContent === fileContent) {
                // If the file didn't change, return any empty array of edits.
                return [];
            }

            const edits = [
                new vscode.TextEdit(
                    new vscode.Range(
                        document.positionAt(0),
                        document.positionAt(fileContent.length),
                    ),
                    formattedContent,
                ),
            ];
            return edits;
        } catch (err) {
            vscode.window.showErrorMessage(`${err}`);
        }
    }
}
