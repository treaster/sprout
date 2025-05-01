# sprout

*Sprout is a work-in-progress. The following is close to working, but it's
not quite there yet. Stay tuned for a working example.*

Sprout is a tool for instantiating a directory of templates by combining them
with a single set of input parameters. The tool does the following steps:
1. Recursively search a directory for input files.
2. When it finds any file, it checks the file extension to determine if the
   file is a template.
   * If no, it copies the file straight across to the corresponding location
     in the output directory.
   * If yes, it executes the template using the shared parameters to generate
     the output content, then writes that to the correponding file in the
     output directory, sans the template file extension.

The intention is to make it easy to generate software project boilerplate code
from a template specification. The input parameters configure various project
options, perhaps including things like "Do you need a Postgres database?" or
"Do you need a Kafka consumer?" The template files are then parameterized
appropriately to include or disclude code blocks corresponding to the options.
If the Postgres option is enabled, then all Postgres boilerplate code is
included. If the options is disabled, then that same code is omitted.

```
go build -o sprout .
./sprout \
    --source-config=example/config.hjson \
    --output=./my_instantiated_template \
    --delete-existing-output
```

or:
```
go run . \
    --source-config=example/config.hjson \
    --output=./my_instantiated_template \
    --delete-existing-output
```

Sprout is inspired by, and borrows a lot of code from, the [incant static site generator](https://github.com/treaster/incant).
