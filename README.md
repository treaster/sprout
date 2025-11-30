# sprout

Sprout is a tool for generating boilerplate software code, by combining
a set of reusable code templates with a single set of input parameters.

The tool does the following steps:
1. Recursively search a directory for input files.
2. When it finds any file, it checks the file extension to determine if the
   file is a template.
   * If no, it copies the file straight across to the corresponding location
     in the output directory.
   * If yes, it executes the template using the shared parameters to generate
     the output content, then writes that to the correponding file in the
     output directory, sans the template file extension.
3. Run an optional post-processing script. This can run any extra installation,
   linting, formatting, or other steps to complete the code generation.

The intention is to make it easy to generate software project boilerplate code
from a template specification.

The input parameters configure various project
options, perhaps including things like "Do you need a Postgres database?" or
"Do you need a Kafka consumer?"

The template files are then parameterized
appropriately to include or disclude code blocks corresponding to the options.

In a hypothetical template, if the Postgres option is enabled, then all
Postgres boilerplate code is included, including plumbing the database
to the code points where it would be used. If that option is disabled, then
that same code is omitted.

Of course, there's no magic here. If a Postgres option is provided, then that
plumbing must be implemented by the template author. But the author can define
the pattern one time, then any number of projects can be generated from the
template, by any number of users. For example, a Sprout template can be a way of
codifying a company's best practices for boilerplate code structure and basic
software patterns, rather than copying a new project from the previous
"most modern" project, then hacking it up to make the next thing.


## Defining a new template
To define a new template, create a directory structure containing a few
different files. See the example/ directory for a simple example.

## config.hjson
The config.hjson file tells the sprout tool how this template should be used
generally, independent of any specific template intantiation or execution. As
a user of a template, you shouldn't need to worry about this too much.

### Fields
####TemplateTypeExt
This represents the file extension for templated files. Files in the project
without this extension will be copied directly to the output directory, with
no changes. The file extension also controls what template language is used by
the project. Options include:
* ".gotempl": The [[text/template][https://pkg.go.dev/text/template]] language provided in the Go standard library.
* ".jet": The [[Jet template language][https://github.com/CloudyKit/jet/blob/master/docs/syntax.md]].
* ".pongo": The [[Pongo2 template language][https://github.com/flosch/pongo2]], which aims to replicate Django.

#### TemplateParamsFile
This names a file which is an example placeholder configuration showing
how an instantation configuration should look. It's an illustration of what
input parameters are available. When you run the sprout tool, this file will
be cloned to your project directory, and you customize it for your specific
project.

By convention, this configuration field is set to "params_template.hjson",
unless there's a good reason to do otherwise.

#### DirsMapping
Sometimes (usually) output directories should be named based on the project perameters.
For example, if someone is invoking your template to define a new service, they
probably want the top-level directory to be something meaningful and specific,
like "rss_reader", rather than something generic, like "service_name".

Additionally, some templates might produce output across multiple different
parts of a code repository. For example, a service's public API definitions
might be separate from the service's underlying implementation.

DirsMapping defines a mapping of template directories to final output
directories, where the final output directories can themselves be templatized
paths. This allows for a name like "service_name" to be mapped at template
execution time to a final directory name like "rss_reader". Multiple mappings
can be specified, enabling code to be generated across the repo, if appropriate.

The map keys represent the search space where sprout looks for input files.
Sprout will recursively search for all files beneath each map key, execute any
templates against the input params, and write the results to the corresponding
DirsMapping value.

#### PostProcessorScript
Often, there are cleanup or follow-on steps that should be performed after
template execution. Getting the templates to format generated code exactly
right can be overly tedious, and this work can often be left to code formatters
and linters.

PostProcessorScript is an optional field which defines a script to run after
the template execution is competed. This script can execute any additional
commands that need to be run.

By default, the post-processor script must be run manually, to allow the user
to manually examine the script for safety before execution. However, if the
template comes from a trusted source, use --autorun-postprocessor on the sprout
command to have Sprout run the post-processor as part of the template execution.


## Using an existing template
To generate code for a new project using a Sprout template:
1. Run the Sprout tool with one of the commands below. This will create the
   project directory, and create a template parameters file with default or
   example values by copying the TemplateParamsFile file to the filepath
   specified by --params.
2. Modify the copied TemplateParamsFile at --params to describe your project.
3. Run the Sprout tool again, using exactly the same command as before. Now
   that the params file is present, the template will be instantiated at
   the --output location, according to the specified parameters.

The example commands below will instantiate the simple example project in the
Sprout repo.


```
# Compile the Sprout tool (do this only once)
go build -o sprout .

# Run the Sprout tool
./sprout \
    --source-config=example/config.hjson \
    --params=./params.hjson \
    --output=./my_instantiated_example
```

or:
```
# Run the Sprout tool with `go run`
go run . \
    --source-config=example/config.hjson \
    --params=./params.hjson \
    --output=./my_instantiated_example
```

## Notes
Sprout is inspired by, and borrows code from, the [incant static site generator](https://github.com/treaster/incant).
