package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// Command implements the behavior of the commands.
type Command struct {
	args []string
	conn net.Conn
}

// handle Executes the command and writes the response.
func (cmd Command) handle() bool {
	switch strings.ToUpper(cmd.args[0]) {
	case "GET":
		return cmd.get()
	case "SET":
		return cmd.set()
	case "DEL":
		return cmd.del()
	case "QUIT":
		return cmd.quit()
	default:
		cmd.conn.Write([]uint8("-ERR unknown command '" + cmd.args[0] + "'\r\n"))
	}
	return true
}

// quit Used in interactive/inline mode, instructs the server to terminate the connection.
func (cmd *Command) quit() bool {
	if len(cmd.args) != 1 {
		cmd.conn.Write([]uint8("-ERR wrong number of arguments for '" + cmd.args[0] + "' command\r\n"))
		return true
	}

	cmd.conn.Write([]uint8("+OK\r\n"))
	return false
}

// del Deletes a key from the cache.
func (cmd *Command) del() bool {
	count := 0
	for _, k := range cmd.args[1:] {
		if _, ok := cache.LoadAndDelete(k); ok {
			// quantity of deleted items
			count++
		}
	}
	cmd.conn.Write([]uint8(fmt.Sprintf(":%d\r\n", count)))
	return true
}

// get Fetches a key from the cache if exists.
func (cmd Command) get() bool {
	if len(cmd.args) != 2 {
		cmd.conn.Write([]uint8("-ERR wrong number of arguments for '" + cmd.args[0] + "' command\r\n"))
		return true
	}

	val, _ := cache.Load(cmd.args[1])
	if val != nil {
		res, _ := val.(string)
		if strings.HasPrefix(res, "\"") {
			res, _ = strconv.Unquote(res)
		}

		// key $length
		cmd.conn.Write([]uint8(fmt.Sprintf("$%d\r\n", len(res))))
		// key value
		cmd.conn.Write(append([]uint8(res), []uint8("\r\n")...))
	} else {
		// key not found returns $-1
		cmd.conn.Write([]uint8("$-1\r\n"))
	}
	return true
}

// set Stores a key and value on the cache. Optionally sets expiration on the key.
func (cmd Command) set() bool {
	if len(cmd.args) < 3 || len(cmd.args) > 6 {
		cmd.conn.Write([]uint8("-ERR wrong number of arguments for '" + cmd.args[0] + "' command\r\n"))
		return true
	}

	if len(cmd.args) > 3 {
		pos := 3
		option := strings.ToUpper(cmd.args[pos])
		switch option {
		case "NX": // only executed if the key doesnt exists
			if _, ok := cache.Load(cmd.args[1]); ok {
				cmd.conn.Write([]uint8("$-1\r\n"))
				return true
			}
			pos++
		case "XX": // only executed if the key exists
			if _, ok := cache.Load(cmd.args[1]); !ok {
				cmd.conn.Write([]uint8("$-1\r\n"))
				return true
			}
			pos++
		}
		if len(cmd.args) > pos {
			if err := cmd.setExpiration(pos); err != nil {
				cmd.conn.Write([]uint8("-ERR " + err.Error() + "\r\n"))
				return true
			}
		}
	}
	// store and return ok
	cache.Store(cmd.args[1], cmd.args[2])
	cmd.conn.Write([]uint8("+OK\r\n"))
	return true
}

// setExpiration Handles expiration when passed as part of the 'set' command.
func (cmd Command) setExpiration(pos int) error {
	option := strings.ToUpper(cmd.args[pos])
	value, _ := strconv.Atoi(cmd.args[pos+1])
	var duration time.Duration
	switch option {
	case "EX":
		duration = time.Second * time.Duration(value)
	case "PX":
		duration = time.Millisecond * time.Duration(value)
	default:
		return fmt.Errorf("expiration option is not valid")
	}
	go func() {
		// sleep for the duretion until its deleted
		// TODO: find better way?
		time.Sleep(duration)
		cache.Delete(cmd.args[1])
	}()
	return nil
}
