import * as vscode from "vscode";
import * as format_provider from "./format_provider";

export function activate(ctx: vscode.ExtensionContext): void {
    ctx.subscriptions.push(
        vscode.languages.registerDocumentFormattingEditProvider("mro",
            new format_provider.MroFormatProvider()));
}
