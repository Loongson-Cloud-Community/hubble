// Code generated by go-swagger; DO NOT EDIT.

// Copyright 2017-2021 Authors of Cilium
// SPDX-License-Identifier: Apache-2.0

package policy

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/cilium/cilium/api/v1/models"
)

// GetIPReader is a Reader for the GetIP structure.
type GetIPReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetIPReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGetIPOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 400:
		result := NewGetIPBadRequest()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 404:
		result := NewGetIPNotFound()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result

	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewGetIPOK creates a GetIPOK with default headers values
func NewGetIPOK() *GetIPOK {
	return &GetIPOK{}
}

/*GetIPOK handles this case with default header values.

Success
*/
type GetIPOK struct {
	Payload []*models.IPListEntry
}

func (o *GetIPOK) Error() string {
	return fmt.Sprintf("[GET /ip][%d] getIpOK  %+v", 200, o.Payload)
}

func (o *GetIPOK) GetPayload() []*models.IPListEntry {
	return o.Payload
}

func (o *GetIPOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetIPBadRequest creates a GetIPBadRequest with default headers values
func NewGetIPBadRequest() *GetIPBadRequest {
	return &GetIPBadRequest{}
}

/*GetIPBadRequest handles this case with default header values.

Invalid request (error parsing parameters)
*/
type GetIPBadRequest struct {
	Payload models.Error
}

func (o *GetIPBadRequest) Error() string {
	return fmt.Sprintf("[GET /ip][%d] getIpBadRequest  %+v", 400, o.Payload)
}

func (o *GetIPBadRequest) GetPayload() models.Error {
	return o.Payload
}

func (o *GetIPBadRequest) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetIPNotFound creates a GetIPNotFound with default headers values
func NewGetIPNotFound() *GetIPNotFound {
	return &GetIPNotFound{}
}

/*GetIPNotFound handles this case with default header values.

No IP cache entries with provided parameters found
*/
type GetIPNotFound struct {
}

func (o *GetIPNotFound) Error() string {
	return fmt.Sprintf("[GET /ip][%d] getIpNotFound ", 404)
}

func (o *GetIPNotFound) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}