package common

const (
	MASTER_LOCK_DIR = "/init/master/"
	JOB_LOCK_KEY = "/cron/lock/"

	JOB_WORKER_DIR = "/cron/workers/"
	JOB_SAVE_DIR   = "/cron/jobs/"
	JOB_DELETE_DIR = "/cron/delete/"
	JOB_KILL_DIR   = "/cron/kill/"

	JOB_EVENT_SAVE   = 1
	JOB_EVENT_DELETE = 2
	JOB_EVENT_KILL   = 3

	TIME_FORMAT = "2006-01-02 15:04:05"
)
