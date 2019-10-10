package bully

// msgType represents the type of messages that can be passed between nodes
type msgType int

// customized messages to be used for communication between nodes
const (
	ELECTION msgType = iota
	OK
	LEADER
	ALIVE
	ACK

	// TODO: add / change message types as needed
)

// Message is used to communicate with the other nodes
// DO NOT MODIFY THIS STRUCT
type Message struct {
	Pid   int
	Round int
	Type  msgType
}

// TrackMessage is used to track replies
type TrackMessage struct {
	Round int
	Pid   int
}

// Bully runs the bully algorithm for leader election on a node
func Bully(pid int, leader int, checkLeader chan bool, comm map[int]chan Message, startRound <-chan int, quit <-chan bool, electionResult chan<- int) {

	// TODO: initialization code
	LastRoundL := 0
	LastRoundE := 0
	CheckForLeader := false
	StartElection := false
	RoundProcessed := -1
	// var MsgList []TrackMessage
	for {

		// quit / crash the program
		if <-quit {
			return
		}

		// start the round
		roundNum := <-startRound
		if pid >= leader && RoundProcessed == -1 {
			// fmt.Println("FAILED NODE ", roundNum, "pid ", pid)
			// fmt.Println("Sending on Election Channel by - ", pid, "Leader = ", leader)
			// electionResult <- leader
			for _, ch := range comm {
				// if index != pid {
				ch <- Message{pid, roundNum, LEADER}

				// }
			}

		}
		select {
		case <-checkLeader:
			CheckForLeader = true
			comm[leader] <- Message{pid, roundNum, ALIVE}
			LastRoundL = roundNum
			// fmt.Println("Checking leader by pid = ", pid, "at Round ", roundNum)

			// SEND LEADER THE MESSAGE
		default:
		}
		MsgList := getMessages(comm[pid], roundNum-1)
		for i := range MsgList {
			NewMsg := MsgList[i]
			if NewMsg.Type == LEADER {
				// fmt.Println("NEW LEADER ", roundNum, pid)
				leader = NewMsg.Pid
				electionResult <- leader
				StartElection = false

			}
			if NewMsg.Type == ALIVE {
				comm[NewMsg.Pid] <- Message{pid, roundNum, ACK}

			}
			if NewMsg.Type == OK {
				StartElection = false

			}
			if NewMsg.Type == ELECTION {

				comm[NewMsg.Pid] <- Message{pid, roundNum, OK}
				if StartElection != true {
					StartElection = true
					LastRoundE = roundNum
					// fmt.Println("Election message Received by ", pid, " Sent by ", NewMsg.Pid)
					for index, ch := range comm {
						if index > pid {
							ch <- Message{pid, roundNum, ELECTION}

						}
					}

				}

				// CheckForLeader = true
				// LastRoundL = roundNum
			}
		}

		if roundNum == LastRoundL+2 {
			if CheckForLeader == true {
				LastRoundMsgs := getMessages(comm[leader], roundNum-1)

				for i := range LastRoundMsgs {
					if LastRoundMsgs[i].Type == ACK {
						CheckForLeader = false
						StartElection = false

					}
				}
				if CheckForLeader == true {
					// fmt.Println("LEADER DEAD ", roundNum, "thread = ", pid)
					StartElection = true
					LastRoundE = roundNum
					for index, ch := range comm {
						if index > pid {
							ch <- Message{pid, roundNum, ELECTION}

						}
					}
					// fmt.Println("Starting Elec ", roundNum, "thread = ", pid, "LASTROUNDE = ", LastRoundE)
				}

			}
		}
		// fmt.Println("Last Round E = ", LastRoundE, " pid ", pid)
		if StartElection == true && LastRoundE+2 == roundNum {
			// fmt.Println("Election! ", roundNum, LastRoundE, "pid ", pid)
			for index := range comm {

				if index > pid {
					LastRoundMsgs := getMessages(comm[index], roundNum-1)
					for i := range LastRoundMsgs {
						if LastRoundMsgs[i].Type == OK {
							StartElection = false
							break
						}

					}
					if StartElection == false {
						break
					}
				}

			}
			if StartElection == true {
				// fmt.Println("PID = ", pid)
				electionResult <- pid
				leader = pid
				// fmt.Println("Sending on Election Channel by ", pid, "Leader = ", leader)

				for index, channel := range comm {
					if index != pid {

						channel <- Message{pid, roundNum, LEADER}

					}

				}
				StartElection = false
				// CheckForLeader = false

			}

		}
		// fmt.Println("Round ", roundNum, "Leader = ", leader, "node = ", pid)
		RoundProcessed = roundNum
		// TODO: bully algorithm code
	}
}

// assumes messages from previous rounds have been read already
func getMessages(msgs chan Message, round int) []Message {
	var result []Message

	// check if there are messages to be read
	if len(msgs) > 0 {
		var m Message

		// read all messages belonging to the corresponding round
		for m = <-msgs; m.Round == round; m = <-msgs {
			result = append(result, m)
			if len(msgs) == 0 {
				break
			}
		}

		// insert back message from any other round
		if m.Round != round {
			msgs <- m
		}
	}
	return result
}
