/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package admin

import (
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sinochem-tech/fabric/common/flogging"
	"github.com/sinochem-tech/fabric/protos/common"
	pb "github.com/sinochem-tech/fabric/protos/peer"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

var logger = flogging.MustGetLogger("server")

type requestValidator interface {
	validate(ctx context.Context, env *common.Envelope) (*pb.AdminOperation, error)
}

// AccessControlEvaluator evaluates whether the creator of the given SignedData
// is eligible of using the admin service
type AccessControlEvaluator interface {
	// Evaluate evaluates the eligibility of the creator of the given SignedData
	// for being serviced by the admin service
	Evaluate(signatureSet []*common.SignedData) error
}

// NewAdminServer creates and returns a Admin service instance.
func NewAdminServer(ace AccessControlEvaluator) *ServerAdmin {
	s := &ServerAdmin{
		v: &validator{
			ace: ace,
		},
	}
	return s
}

// ServerAdmin implementation of the Admin service for the Peer
type ServerAdmin struct {
	v requestValidator
}

func (s *ServerAdmin) GetStatus(ctx context.Context, env *common.Envelope) (*pb.ServerStatus, error) {
	if _, err := s.v.validate(ctx, env); err != nil {
		return nil, err
	}
	status := &pb.ServerStatus{Status: pb.ServerStatus_STARTED}
	logger.Debugf("returning status: %s", status)
	return status, nil
}

func (s *ServerAdmin) StartServer(ctx context.Context, env *common.Envelope) (*pb.ServerStatus, error) {
	if _, err := s.v.validate(ctx, env); err != nil {
		return nil, err
	}
	status := &pb.ServerStatus{Status: pb.ServerStatus_STARTED}
	logger.Debugf("returning status: %s", status)
	return status, nil
}

func (s *ServerAdmin) GetModuleLogLevel(ctx context.Context, env *common.Envelope) (*pb.LogLevelResponse, error) {
	op, err := s.v.validate(ctx, env)
	if err != nil {
		return nil, err
	}
	request := op.GetLogReq()
	if request == nil {
		return nil, errors.New("request is nil")
	}
	logLevelString := flogging.GetModuleLevel(request.LogModule)
	logResponse := &pb.LogLevelResponse{LogModule: request.LogModule, LogLevel: logLevelString}
	return logResponse, nil
}

func (s *ServerAdmin) SetModuleLogLevel(ctx context.Context, env *common.Envelope) (*pb.LogLevelResponse, error) {
	op, err := s.v.validate(ctx, env)
	if err != nil {
		return nil, err
	}
	request := op.GetLogReq()
	if request == nil {
		return nil, errors.New("request is nil")
	}
	logLevelString, err := flogging.SetModuleLevel(request.LogModule, request.LogLevel)
	logResponse := &pb.LogLevelResponse{LogModule: request.LogModule, LogLevel: logLevelString}
	return logResponse, err
}

func (s *ServerAdmin) RevertLogLevels(ctx context.Context, env *common.Envelope) (*empty.Empty, error) {
	if _, err := s.v.validate(ctx, env); err != nil {
		return nil, err
	}
	err := flogging.RevertToPeerStartupLevels()
	return &empty.Empty{}, err
}
