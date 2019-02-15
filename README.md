# WinOperator Agent

This is based on the [GO Windows service example program](https://godoc.org/golang.org/x/sys/windows/svc/example) provided by the GO Project. 
It is a project shell to create a Windows service.

## Getting Started

The program compiles and runs on GO 1.8.  The generated executable accepts a single parameter.  The parameter values include:

* debug - runs the program from the command-line
* install - installs a windows service
* remove - removes the windows service
* start
* stop
* pause
* continue

## Installing and Updating a Service

After compiling an executable, the service can be installed from an Administrative command prompt.  Typing

    YourExecutable.EXE install 

will install the service.

To update the service, stop the service, replace the executble and restart the service.

The service can be removed from an Administrative command prompt by typing:

    YourExecutable.EXE remove 


## Advanced
### Logging
The `server` struct exposes a `winlog` variable that is a logger.  This will write to the console when running interactively and to the Winodws Application Event Log when running as a service.  I typically use this for any service errors, start and stop notification, and any issues reading configuration or setting up logging.

### Other 
This uses the [GO error wrapper package]("github.com/pkg/errors").  You can easily 
remove it if you prefer not to use it.










