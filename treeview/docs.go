// Package treeview draws a pipeline run as a tree of jobs, steps and commands
// on the terminal. Builder and ExecutionTree build the nodes from a pipeline,
// Node holds the state the runner updates while the run goes on, Renderer
// writes a tree as text and Trimmer fits the labels to the terminal width.
// Display redraws the tree in place on a terminal and prints it once,
// statically, on a pipe or a file.
package treeview
