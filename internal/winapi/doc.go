// Package winapi encapsula las llamadas nativas de Win32 que usa mapadisco
// (enumeración de directorios, espacio en disco, consola, elevación UAC).
// Es la única capa que habla directamente con Windows; el resto del programa
// depende de ella, no de syscall.
package winapi
