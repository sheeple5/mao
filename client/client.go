package main

import (
	"bufio"
	"fmt"
	"net"
)

func main() {
	var choice string
	fmt.Print("Press Y to join the game: ")
	fmt.Scan(&choice)

	if choice == "Y" {
		conn, err := net.Dial("tcp", "localhost:9090")
		if err != nil {
			panic(err)
		}
		defer conn.Close()

		writer := bufio.NewWriter(conn)
		_, err = writer.WriteString("{'joinGame': 'ABCD'}\n")
		if err != nil {
			panic(err)
		}

		err = writer.Flush()
		if err != nil {
			panic(err)
		}

		netData, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			panic(err)
		}
		fmt.Println(netData)

	}
}
