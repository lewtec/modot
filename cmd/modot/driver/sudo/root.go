package sudo

import ()

type Command struct {
	Add     *Add
	Approve *Approve
	List    *List
	Reject  *Reject
}

func (Command) Description() string {
	return "Manage pending privileged commands"
}
