// Package usvc is an opinionated
// (read: not very well implemented)
// micro service framework(yuck, but boilerplate)
//
// Hello world:
//
//      c := usvc.NewConfig(flag.CommandLine)
//      flag.Parse()
//      usvc.Run(usvc.SignalContext(), usvc.NewServerSimple(c))
//
// With custom things:
//
//     var foo string
//     fs := flag.NewFlagSet(args[0], flag.ExitOnError)
//     fs.StringVar(&foo, "foo", "bar", "foobar")
//     svc := usvc.NewServerSimple(usvc.NewConfig(fs))
//     fs.Parse()
//
//     svc.Mux.HandleFunc("/foo", func(w http.ResponseWriter, r *http.Request){
//         w.Write([]byte(foo))
//     })
//
//     usvc.Run(usvc.SignalContext(), svc)
package usvc
