package request

import (
	"context"
	"errors"

	gc "github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/common"
	pb "github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/stub/cbspider"
)

// ===== [ Constants and Variables ] =====

// ===== [ Types ] =====

// ===== [ Implementations ] =====

// CreateSecurity - Security 생성
func (r *CCMRequest) CreateSecurity() (string, error) {
	// 입력데이터 검사
	if r.InData == "" {
		return "", errors.New("input data required")
	}

	// 입력데이터 언마샬링
	var item pb.SecurityCreateRequest
	err := gc.ConvertToMessage(r.InType, r.InData, &item)
	if err != nil {
		return "", err
	}

	// 서버에 요청
	ctx, cancel := context.WithTimeout(context.Background(), r.Timeout)
	defer cancel()

	resp, err2 := r.Client.CreateSecurity(ctx, &item)
	if err2 != nil {
		return "", err2
	}

	// 결과값 마샬링
	return gc.ConvertToOutput(r.OutType, &resp.Item)
}

// ListSecurity - Security 목록
func (r *CCMRequest) ListSecurity() (string, error) {
	// 입력데이터 검사
	if r.InData == "" {
		return "", errors.New("input data required")
	}

	// 입력데이터 언마샬링
	var item pb.SecurityAllQryRequest
	err := gc.ConvertToMessage(r.InType, r.InData, &item)
	if err != nil {
		return "", err
	}

	// 서버에 요청
	ctx, cancel := context.WithTimeout(context.Background(), r.Timeout)
	defer cancel()

	resp, err := r.Client.ListSecurity(ctx, &item)

	if err != nil {
		return "", err
	}

	// 결과값 마샬링
	return gc.ConvertToOutput(r.OutType, &resp)
}

// GetSecurity - Security 조회
func (r *CCMRequest) GetSecurity() (string, error) {
	// 입력데이터 검사
	if r.InData == "" {
		return "", errors.New("input data required")
	}

	// 입력데이터 언마샬링
	var item pb.SecurityQryRequest
	err := gc.ConvertToMessage(r.InType, r.InData, &item)
	if err != nil {
		return "", err
	}

	// 서버에 요청
	ctx, cancel := context.WithTimeout(context.Background(), r.Timeout)
	defer cancel()

	resp, err2 := r.Client.GetSecurity(ctx, &item)
	if err2 != nil {
		return "", err2
	}

	// 결과값 마샬링
	return gc.ConvertToOutput(r.OutType, &resp.Item)
}

// DeleteSecurity - Security 삭제
func (r *CCMRequest) DeleteSecurity() (string, error) {
	// 입력데이터 검사
	if r.InData == "" {
		return "", errors.New("input data required")
	}

	// 입력데이터 언마샬링
	var item pb.SecurityQryRequest
	err := gc.ConvertToMessage(r.InType, r.InData, &item)
	if err != nil {
		return "", err
	}

	// 서버에 요청
	ctx, cancel := context.WithTimeout(context.Background(), r.Timeout)
	defer cancel()

	resp, err2 := r.Client.DeleteSecurity(ctx, &item)
	if err2 != nil {
		return "", err2
	}

	// 결과값 마샬링
	return gc.ConvertToOutput(r.OutType, &resp)
}

// ListAllSecurity - 관리 Security 목록
func (r *CCMRequest) ListAllSecurity() (string, error) {
	// 입력데이터 검사
	if r.InData == "" {
		return "", errors.New("input data required")
	}

	// 입력데이터 언마샬링
	var item pb.SecurityAllQryRequest
	err := gc.ConvertToMessage(r.InType, r.InData, &item)
	if err != nil {
		return "", err
	}

	// 서버에 요청
	ctx, cancel := context.WithTimeout(context.Background(), r.Timeout)
	defer cancel()

	resp, err := r.Client.ListAllSecurity(ctx, &item)

	if err != nil {
		return "", err
	}

	// 결과값 마샬링
	return gc.ConvertToOutput(r.OutType, &resp)
}

// DeleteCSPSecurity - 관리 Security 삭제
func (r *CCMRequest) DeleteCSPSecurity() (string, error) {
	// 입력데이터 검사
	if r.InData == "" {
		return "", errors.New("input data required")
	}

	// 입력데이터 언마샬링
	var item pb.CSPSecurityQryRequest
	err := gc.ConvertToMessage(r.InType, r.InData, &item)
	if err != nil {
		return "", err
	}

	// 서버에 요청
	ctx, cancel := context.WithTimeout(context.Background(), r.Timeout)
	defer cancel()

	resp, err2 := r.Client.DeleteCSPSecurity(ctx, &item)
	if err2 != nil {
		return "", err2
	}

	// 결과값 마샬링
	return gc.ConvertToOutput(r.OutType, &resp)
}

// ===== [ Private Functions ] =====

// ===== [ Public Functions ] =====
