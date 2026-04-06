//go:build windows

package exiftool

import (
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	globalJobOnce sync.Once
	globalJob     windows.Handle
	globalJobErr  error
)

func initJobObject() {
	globalJobOnce.Do(func() {
		job, err := windows.CreateJobObject(nil, nil)
		if err != nil {
			globalJobErr = err
			return
		}

		var info windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
		info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE

		_, err = windows.SetInformationJobObject(
			job,
			windows.JobObjectExtendedLimitInformation,
			uintptr(unsafe.Pointer(&info)),
			uint32(unsafe.Sizeof(info)),
		)
		if err != nil {
			windows.CloseHandle(job)
			globalJobErr = err
			return
		}

		globalJob = job
	})
}

func assignToJob(pid int) error {
	initJobObject()
	if globalJobErr != nil {
		return globalJobErr
	}

	proc, err := windows.OpenProcess(
		windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE,
		false, uint32(pid),
	)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(proc)

	return windows.AssignProcessToJobObject(globalJob, proc)
}
