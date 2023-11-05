package storage

type Options struct {
	redisAddress                               string
	dbHost, dbUser, dbPassword, dbName, dbPort string
}

func DefaultOptions() *Options {
	return &Options{
		redisAddress: "0.0.0.0:6379",
		dbHost:       "0.0.0.0",
		dbUser:       "postgres",
		dbPassword:   "pswd",
		dbName:       "sensors",
		dbPort:       "5432",
	}
}

type Option func(opt *Options)

func WithRedisAddress(addr string) Option {
	return func(opt *Options) {
		opt.redisAddress = addr
	}
}

func WithDbHost(host string) Option {
	return func(opt *Options) {
		opt.dbHost = host
	}
}

func WithDbPort(port string) Option {
	return func(opt *Options) {
		opt.dbPort = port
	}
}

func WithDbUser(user string) Option {
	return func(opt *Options) {
		opt.dbUser = user
	}
}

func WithDbPassword(pswd string) Option {
	return func(opt *Options) {
		opt.dbPassword = pswd
	}
}

func WithDbName(name string) Option {
	return func(opt *Options) {
		opt.dbName = name
	}
}
