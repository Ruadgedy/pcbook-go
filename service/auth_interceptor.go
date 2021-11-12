package service

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"log"
)

// AuthInterceptor is a server interceptor for authentication and authorization
type AuthInterceptor struct {
	jwtManager *JWTManager
	accessibleRoles map[string][]string // 存储每个RPC方法与能够访问它的角色：key是RPC方法名字，value是角色切片类型
}

func NewAuthInterceptor(jwtManager *JWTManager, accessibleRoles map[string][]string) *AuthInterceptor {
	return &AuthInterceptor{jwtManager,accessibleRoles}
}

// Unary returns a server interceptor function to authentication and authorize unary RPC
func (interceptor *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	)(resp interface{}, err error){
		log.Println("--> unary interceptor: ", info.FullMethod)

		// 验证是否有权限
		err = interceptor.authorize(ctx, info.FullMethod)
		if err != nil {
			return nil, err
		}

		return handler(ctx,req)
	}
}

// Stream returns a server interceptor function to authentication and authorize stream RPC
func (interceptor *AuthInterceptor) Stream() grpc.StreamServerInterceptor {
	return func (
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		log.Println("--> stream interceptor: ",info.FullMethod)

		// 验证是否有权限
		err := interceptor.authorize(stream.Context(), info.FullMethod)
		if err != nil {
			return err
		}

		return handler(srv, stream)
	}
}

func (interceptor *AuthInterceptor) authorize(ctx context.Context, method string) error {
	// 拿到该RPC方法所需要的权限
	accessibleRoles, ok := interceptor.accessibleRoles[method]
	if !ok {
		// 权限列表中没有key，则说明任何人都可以访问
		return nil
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok{
		return status.Errorf(codes.Unauthenticated,"metadata is not provided")
	}

	values := md["authorization"]
	if len(values) == 0 {
		return status.Errorf(codes.Unauthenticated,"authorization token is not provided")
	}

	accessToken := values[0]
	claims, err := interceptor.jwtManager.Verify(accessToken)
	if err != nil {
		return status.Errorf(codes.Unauthenticated,"access token is invalid: %v" ,err)
	}

	// 遍历所需权限，判断用户是否有该权限
	for _, role := range accessibleRoles {
		if role == claims.Role {
			return nil
		}
	}

	return status.Errorf(codes.PermissionDenied, "no permission to access this RPC")
}