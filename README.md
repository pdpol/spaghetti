# spaghetti
Search Python function/method definitions and usage from commandline

![Spaghetti](https://images.duckduckgo.com/iu/?u=http%3A%2F%2F25.media.tumblr.com%2Ftumblr_m39b71JuMu1rrftcdo1_500.gif&f=1)

## TODO
* Support more languages
* Smarater detection of the end of a method/function
* Support config files for projects
* Support root directory arg
* Regex for exclude_patterns flag
* Framework-aware searching. I exclude `urls.py` from my Django project search only because it breaks my currently limited
functionality. Should recognize routes pointing to the method searched for instead of... break.

## Wat is this
Spaghetti walks a directory tree recursively and searches for a Python function/method name. The definition and blocks of code
surrounding calls are printed on stdout.

I wrote this because I find myself running `grep -r 'method_name' .` pretty often and instead of popping open every file,
I'd rather just see the whole call in the terminal.

## Usage
* For now, `cd` to the root directory, support for passing a root arg, and config file to be done soon!
```
spaghetti [flags] method_name
```

### Supported flags/args
* --exclude_patterns : comma-separated list of strings to ignore when walking the dirctory path
