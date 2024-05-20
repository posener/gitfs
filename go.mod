module github.com/posener/gitfs

go 1.12

require (
	github.com/go-git/go-billy/v5 v5.5.0
	github.com/go-git/go-git/v5 v5.12.0
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/kr/fs v0.1.0
	github.com/pkg/errors v0.9.1
	github.com/posener/diff v0.0.1
	github.com/stretchr/testify v1.9.0
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	golang.org/x/tools v0.13.0
	google.golang.org/appengine v1.6.1 // indirect
)

replace rsc.io/diff => github.com/posener/diff v0.0.0-20190808172948-eff7f6d9b194
