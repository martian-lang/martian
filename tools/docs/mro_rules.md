<!-- Generated with Stardoc: http://skydoc.bazel.build -->

<a id="#mrf_test"></a>

## mrf_test

<pre>
mrf_test(<a href="#mrf_test-name">name</a>, <a href="#mrf_test-mrf_flags">mrf_flags</a>, <a href="#mrf_test-script_name">script_name</a>, <a href="#mrf_test-srcs">srcs</a>)
</pre>

Runs `mro format` on the given files, and fails if there are any differences.

**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="mrf_test-name"></a>name |  A unique name for this target.   | <a href="https://bazel.build/docs/build-ref.html#name">Name</a> | required |  |
| <a id="mrf_test-mrf_flags"></a>mrf_flags |  Flags to pass to <code>mro format</code>.   | List of strings | optional | ["--includes"] |
| <a id="mrf_test-script_name"></a>script_name |  The name for the script file.   | String | optional | "" |
| <a id="mrf_test-srcs"></a>srcs |  The mro files to format.   | <a href="https://bazel.build/docs/build-ref.html#labels">List of labels</a> | required |  |


<a id="#mro_library"></a>

## mro_library

<pre>
mro_library(<a href="#mro_library-name">name</a>, <a href="#mro_library-data">data</a>, <a href="#mro_library-deps">deps</a>, <a href="#mro_library-mropath">mropath</a>, <a href="#mro_library-srcs">srcs</a>)
</pre>


A rule for collecting an mro file and its dependencies.

Transitively collects `MROPATH` from other `mro_library` dependencies,
as well as `PYTHONPATH` from any python dependencies.

`.mro` files should be supplied in `srcs`.  Other `mro` targets included
by files in `srcs`, as well as the targets implementing stages declared in the
`srcs`, should be included in `deps`.


**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="mro_library-name"></a>name |  A unique name for this target.   | <a href="https://bazel.build/docs/build-ref.html#name">Name</a> | required |  |
| <a id="mro_library-data"></a>data |  Data dependencies for top-level calls defined in srcs.   | <a href="https://bazel.build/docs/build-ref.html#labels">List of labels</a> | optional | [] |
| <a id="mro_library-deps"></a>deps |  Included mro libraries and stage code for stages defined in srcs.   | <a href="https://bazel.build/docs/build-ref.html#labels">List of labels</a> | optional | [] |
| <a id="mro_library-mropath"></a>mropath |  A path to add to the <code>MROPATH</code> for targets which depend on this <code>mro_library</code>, relative to the package.<br><br>The current default is '.', meaning the package directory.  This will change in the future, after some time for migration, to be empty instead.   | String | optional | "." |
| <a id="mro_library-srcs"></a>srcs |  .mro source file(s) for this library target.   | <a href="https://bazel.build/docs/build-ref.html#labels">List of labels</a> | optional | [] |


<a id="#mro_test"></a>

## mro_test

<pre>
mro_test(<a href="#mro_test-name">name</a>, <a href="#mro_test-flags">flags</a>, <a href="#mro_test-srcs">srcs</a>, <a href="#mro_test-strict">strict</a>)
</pre>


Runs `mro check` on the `mro_library` targets supplied in `srcs`.


**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="mro_test-name"></a>name |  A unique name for this target.   | <a href="https://bazel.build/docs/build-ref.html#name">Name</a> | required |  |
| <a id="mro_test-flags"></a>flags |  Additional flags to pass to <code>mro check</code>   | List of strings | optional | [] |
| <a id="mro_test-srcs"></a>srcs |  -   | <a href="https://bazel.build/docs/build-ref.html#labels">List of labels</a> | optional | [] |
| <a id="mro_test-strict"></a>strict |  Run <code>mro check</code> with the <code>--strict</code> flag.   | Boolean | optional | True |


<a id="#mro_tool_runner"></a>

## mro_tool_runner

<pre>
mro_tool_runner(<a href="#mro_tool_runner-name">name</a>, <a href="#mro_tool_runner-flags">flags</a>, <a href="#mro_tool_runner-script_name">script_name</a>, <a href="#mro_tool_runner-srcs">srcs</a>, <a href="#mro_tool_runner-subcommand">subcommand</a>)
</pre>

Runs the `mro` tool, possibly with a subcommand, with the given mro files (if any) as arguments.

**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="mro_tool_runner-name"></a>name |  A unique name for this target.   | <a href="https://bazel.build/docs/build-ref.html#name">Name</a> | required |  |
| <a id="mro_tool_runner-flags"></a>flags |  Flags to pass to the <code>mro</code> subcommand.   | List of strings | optional | [] |
| <a id="mro_tool_runner-script_name"></a>script_name |  The name for the script file.   | String | optional | "" |
| <a id="mro_tool_runner-srcs"></a>srcs |  The mro files to give as arguments to the tool.   | <a href="https://bazel.build/docs/build-ref.html#labels">List of labels</a> | optional | [] |
| <a id="mro_tool_runner-subcommand"></a>subcommand |  The subcommand for the <code>mro</code> tool, e.g. <code>format</code>, <code>check</code>, <code>graph</code>, <code>edit</code>.   | String | required |  |


<a id="#mrf_runner"></a>

## mrf_runner

<pre>
mrf_runner(<a href="#mrf_runner-mrf_flags">mrf_flags</a>, <a href="#mrf_runner-srcs">srcs</a>, <a href="#mrf_runner-kwargs">kwargs</a>)
</pre>

A convenience wrapper for running `mro format` on a set of files.

**PARAMETERS**


| Name  | Description | Default Value |
| :------------- | :------------- | :------------- |
| <a id="mrf_runner-mrf_flags"></a>mrf_flags |  Flags to pass to <code>mro format</code>.   |  <code>["--rewrite", "--includes"]</code> |
| <a id="mrf_runner-srcs"></a>srcs |  The set of mro files or targets to format.  If an <code>mro_library</code>   target is supplied, its transitive dependencies will be formatted   as well.   |  <code>[]</code> |
| <a id="mrf_runner-kwargs"></a>kwargs |  Any other attributes to pass through to the underlying rule,   including for example <code>name</code> and <code>visibility</code>.   |  none |


