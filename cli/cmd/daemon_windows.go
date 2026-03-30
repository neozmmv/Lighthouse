package cmd

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

func daemonSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		// DETACHED_PROCESS | CREATE_NO_WINDOW
		// detaches the daemon from the parent terminal and hides console windows
		CreationFlags: 0x00000008 | 0x08000000,
	}
}

// createJobObject creates a Windows Job Object that kills all child
// processes when the daemon exits
func createJobObject() (windows.Handle, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, err
	}

	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE

	_, err = windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	)
	if err != nil {
		windows.CloseHandle(job)
		return 0, err
	}

	return job, nil
}

// assignToJob assigns a process to the Job Object
func assignToJob(job windows.Handle, pid int) error {
	h, err := windows.OpenProcess(windows.PROCESS_ALL_ACCESS, false, uint32(pid))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)
	return windows.AssignProcessToJobObject(job, h)
}
