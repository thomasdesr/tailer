package tailer

// func (t *File) notifyForRotations()

// func (t *File) handleFileEvent(ev fsnotify.Event) fileAction {
// 	switch {
// 	case isOp(ev, fsnotify.Create):
// 		// new file created with the same name
// 		return reopenFile

// 	case isOp(ev, fsnotify.Write):
// 		// On write, check to see if the file has been truncated
// 		// If not, insure the bufio buffer is full
// 		switch f.checkForTruncate() {
// 		case true:
// 			return reopenFile
// 		case false:
// 			// COmeback and re-add this once we setup something other than polling for the fill
// 			return readFile
// 		}

// 	case isOp(ev, fsnotify.Remove), isOp(ev, fsnotify.Rename):
// 		// wait for a new file to be created
// 		return noop

// 	case isOp(ev, fsnotify.Chmod):
// 		// Modified time on the file changed, noop
// 		return noop

// 	default:
// 		panic(fmt.Sprintf("recieved unknown fsnotify event: %#v", ev))
// 	}
// }
