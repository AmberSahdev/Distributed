/*
Your centralized logger should
  1. listen on a port, specified on a command line,
  2. allow nodes to connect to it and start sending it events.
  3. print out the events, along with the name of the node sending
      the events, to standard out.
(If you want to include diagnostic messages, make sure those are sent to stderr)

You do not need to implement an explicit failure detector;
it is sufficient to create a TCP connection from the nodes to the logger and
have the logger report when it closes.
*/
