import * as vscode from "vscode";
import * as format_provider from "./format_provider";

export function activate(context: vscode.ExtensionContext) {
    vscode.languages.registerDocumentFormattingEditProvider("mro",
        new format_provider.MroFormatProvider());
}
