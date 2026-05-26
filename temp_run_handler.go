func (handler *RunHandler) ServeHTTP(w http.ResponseWriter, r *http.Request, runningMu *sync.Mutex, running map[string]*exec.Cmd) {
	port := utils.PickFreePort()

	cmd := handler.ProjectService.StartNodeApplication(utils.WorkDir, port)

	runningMu.Lock()
	running[id] = cmd
	runningMu.Unlock()

	// return an endpoint to access it
	// (we'll run a reverse proxy in a next step if you want)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"id":"%s","port":%d,"message":"started"}`, id, port)

	// Let it keep running; on crash, user will see it later.
	// (Optionally you can capture logs to file — easy next improvement.)
	go func() {
		err := cmd.Wait()
		if err != nil {
			log.Printf("project %s exited: %v", id, err)
		}
	}()
}