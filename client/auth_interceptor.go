package client

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"log"
	"time"
)

// 我们向server发送请求之前，拦截下所有请求，舔加上access token再放行

//AuthInterceptor is a client interceptor for authentication
type AuthInterceptor struct {
	authClient *AuthClient
	authMethods map[string]bool
	accessToken string
}

// Unary returns a client interceptor to authenticate unary RPC
func (interceptor *AuthInterceptor) Unary() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		log.Printf("--> unary interceptor: %s", method)

		if interceptor.authMethods[method] {
			// 如果方法需要认证，则在context中加入token
			return invoker(interceptor.attachToken(ctx),method,req,reply,cc,opts...)
		}
		return invoker(ctx,method,req,reply,cc,opts...)
	}
}

// Stream returns a client interceptor to authenticate stream RPC
func (interceptor *AuthInterceptor) Stream() grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (stream grpc.ClientStream, err error) {
		log.Printf("--> stream interceptor: %s",method)

		if interceptor.authMethods[method] {
			return streamer(interceptor.attachToken(ctx),desc,cc,method,opts...)
		}

		return streamer(ctx,desc,cc,method,opts...)
	}
}


// 定时刷新token
func (interceptor *AuthInterceptor) scheduleRefreshToken(refreshDuration time.Duration) error{
	err := interceptor.refreshToken()
	if err != nil {
		return err
	}

	go func() {
		wait := refreshDuration
		for {
			time.Sleep(wait)
			err := interceptor.refreshToken()
			if err != nil {
				wait = time.Second
			}else {
				wait = refreshDuration
			}
		}
	}()

	return nil
}

// 执行刷新token操作
func (interceptor *AuthInterceptor) refreshToken() error{
	accessToken, err := interceptor.authClient.Login()
	if err != nil {
		return err
	}

	interceptor.accessToken = accessToken
	log.Printf("token refreshed: %v\n",accessToken)

	return nil
}

// 将accessToken 附加到context
func (interceptor *AuthInterceptor) attachToken(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx,"authorization",interceptor.accessToken)
}

//NewAuthInterceptor creates a new auth interceptor
func NewAuthInterceptor(
	authClient *AuthClient,
	authMethods map[string]bool,
	refreshDuration time.Duration,
) (*AuthInterceptor,error){
	interceptor := &AuthInterceptor{
		authClient:  authClient,
		authMethods: authMethods,
	}

	err := interceptor.scheduleRefreshToken(refreshDuration)
	if err != nil {
		return nil, err
	}

	return interceptor,nil
}
