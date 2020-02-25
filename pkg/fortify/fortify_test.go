package fortify

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	ff "github.com/piper-validation/fortify-client-go/fortify"
	"github.com/piper-validation/fortify-client-go/models"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"

	piperHttp "github.com/SAP/jenkins-library/pkg/http"
)

func spinUpServer(f func(http.ResponseWriter, *http.Request)) (*SystemInstance, *httptest.Server) {
	server := httptest.NewServer(http.HandlerFunc(f))

	parts := strings.Split(server.URL, "://")
	client := ff.NewHTTPClientWithConfig(strfmt.Default, &ff.TransportConfig{
		Host:     parts[1],
		Schemes:  []string{parts[0]},
		BasePath: ""},
	)

	httpClient := &piperHttp.Client{}
	httpClientOptions := piperHttp.ClientOptions{Token: "test2456", Timeout: 60 * time.Second}
	httpClient.SetOptions(httpClientOptions)

	sys := NewSystemInstanceForClient(client, httpClient, server.URL, "test2456", 60*time.Second)
	return sys, server
}

func TestNewSystemInstance(t *testing.T) {
	sys := NewSystemInstance("https://some.fortify.host.com/ssc", "api/v1", "akjhskjhks", 10*time.Second)
	assert.IsType(t, ff.Fortify{}, *sys.client, "Expected to get a Fortify client instance")
	assert.IsType(t, piperHttp.Client{}, *sys.httpClient, "Expected to get a HTTP client instance")
	assert.IsType(t, logrus.Entry{}, *sys.logger, "Expected to get a logrus entry instance")
	assert.Equal(t, 10*time.Second, sys.timeout, "Expected different timeout value")
	assert.Equal(t, "akjhskjhks", sys.token, "Expected different token value")
}

func TestGetProjectByName(t *testing.T) {
	// Start a local HTTP server
	sys, server := spinUpServer(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/projects" && req.URL.RawQuery == "fulltextsearch=true&q=name%3Dpython-test" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			rw.Write([]byte(
				`{"data": [{"_href": "https://fortify/ssc/api/v1/projects/4711","createdBy": "someUser","name": "python-test",
				"description": "","id": 4711,"creationDate": "2018-12-03T06:29:38.197+0000","issueTemplateId": "dasdasdasdsadasdasdasdasdas"}],
				"count": 1,"responseCode": 200,"links": {"last": {"href": "https://fortify/ssc/api/v1/projects?q=name%A3python-test&start=0"},
				"first": {"href": "https://fortify/ssc/api/v1/projects?q=name%A3python-test&start=0"}}}`))
			return
		}
		if req.URL.Path == "/projects" && req.URL.RawQuery == "fulltextsearch=true&q=name%3Dpython-empty" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			rw.Write([]byte(
				`{"data": [],"count": 0,"responseCode": 404,"links": {}}`))
			return
		}
		if req.URL.Path == "/projects" && req.URL.RawQuery == "fulltextsearch=true&q=name%3Dpython-error" {
			rw.WriteHeader(400)
			return
		}
	})
	// Close the server when test finishes
	defer server.Close()

	t.Run("test success", func(t *testing.T) {
		result, err := sys.GetProjectByName("python-test")
		assert.NoError(t, err, "GetProjectByName call not successful")
		assert.Equal(t, "python-test", strings.ToLower(*result.Name), "Expected to get python-test")
	})

	t.Run("test empty", func(t *testing.T) {
		_, err := sys.GetProjectByName("python-empty")
		assert.Error(t, err, "Expected error but got success")
	})

	t.Run("test error", func(t *testing.T) {
		_, err := sys.GetProjectByName("python-error")
		assert.Error(t, err, "Expected error but got success")
	})
}

func TestGetProjectVersionDetailsByNameAndProjectID(t *testing.T) {
	// Start a local HTTP server
	sys, server := spinUpServer(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/projects/4711/versions" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			rw.Write([]byte(
				`{"data":[{"latestScanId":null,"serverVersion":17.2,"tracesOutOfDate":false,"attachmentsOutOfDate":false,"description":"",
				"project":{"id":4711,"name":"python-test","description":"","creationDate":"2018-12-03T06:29:38.197+0000","createdBy":"someUser",
				"issueTemplateId":"dasdasdasdsadasdasdasdasdas"},"sourceBasePath":null,"mode":"BASIC","masterAttrGuid":"sddasdasda","obfuscatedId":null,
				"id":10172,"customTagValuesAutoApply":null,"issueTemplateId":"dasdasdasdsadasdasdasdasdas","loadProperties":null,"predictionPolicy":null,
				"bugTrackerPluginId":null,"owner":"admin","_href":"https://fortify.mo.sap.corp/ssc/api/v1/projectVersions/10172",
				"committed":true,"bugTrackerEnabled":false,"active":true,"snapshotOutOfDate":false,"issueTemplateModifiedTime":1578411924701,
				"securityGroup":null,"creationDate":"2018-02-09T16:59:41.297+0000","refreshRequired":false,"issueTemplateName":"someTemplate",
				"migrationVersion":null,"createdBy":"admin","name":"0","siteId":null,"staleIssueTemplate":false,"autoPredict":null,
				"currentState":{"id":10172,"committed":true,"attentionRequired":false,"analysisResultsExist":true,"auditEnabled":true,
				"lastFprUploadDate":"2018-02-09T16:59:53.497+0000","extraMessage":null,"analysisUploadEnabled":true,"batchBugSubmissionExists":false,
				"hasCustomIssues":false,"metricEvaluationDate":"2018-03-10T00:02:45.553+0000","deltaPeriod":7,"issueCountDelta":0,"percentAuditedDelta":0.0,
				"criticalPriorityIssueCountDelta":0,"percentCriticalPriorityIssuesAuditedDelta":0.0},"assignedIssuesCount":0,"status":null}],
				"count":1,"responseCode":200,"links":{"last":{"href":"https://fortify.mo.sap.corp/ssc/api/v1/projects/4711/versions?start=0"},
				"first":{"href":"https://fortify.mo.sap.corp/ssc/api/v1/projects/4711/versions?start=0"}}}`))
			return
		}
		if req.URL.Path == "/projects/777/versions" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			rw.Write([]byte(
				`{"data": [],"count": 0,"responseCode": 404,"links": {}}`))
			return
		}
		if req.URL.Path == "/projects/999/versions" {
			rw.WriteHeader(500)
			return
		}
	})

	// Close the server when test finishes
	defer server.Close()

	t.Run("test success", func(t *testing.T) {
		result, err := sys.GetProjectVersionDetailsByNameAndProjectID(4711, "0")
		assert.NoError(t, err, "GetProjectVersionDetailsByNameAndProjectID call not successful")
		assert.Equal(t, "0", *result.Name, "Expected to get project version with different name")
	})

	t.Run("test empty", func(t *testing.T) {
		_, err := sys.GetProjectVersionDetailsByNameAndProjectID(777, "python-empty")
		assert.Error(t, err, "Expected error but got success")
	})

	t.Run("test HTTP error", func(t *testing.T) {
		_, err := sys.GetProjectVersionDetailsByNameAndProjectID(999, "python-http-error")
		assert.Error(t, err, "Expected error but got success")
	})
}

func TestGetProjectVersionAttributesByID(t *testing.T) {
	// Start a local HTTP server
	sys, server := spinUpServer(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/projectVersions/4711/attributes" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			rw.Write([]byte(
				`{"data": [{"_href": "https://fortify.mo.sap.corp/ssc/api/v1/projectVersions/4711/attributes/4712","attributeDefinitionId": 31,
				"values": null,"guid": "gdgfdgfdgfdgfd","id": 4712,"value": "abcd"}],"count": 8,"responseCode": 200}`))
			return
		}
		if req.URL.Path == "/projectVersions/777/attributes" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			rw.Write([]byte(
				`{"data": [],"count": 0,"responseCode": 404,"links": {}}`))
			return
		}
		if req.URL.Path == "/projectVersions/999/attributes" {
			rw.WriteHeader(500)
			return
		}
	})

	// Close the server when test finishes
	defer server.Close()

	t.Run("test success", func(t *testing.T) {
		result, err := sys.GetProjectVersionAttributesByID(4711)
		assert.NoError(t, err, "GetProjectVersionAttributesByID call not successful")
		assert.Equal(t, "abcd", *result[0].Value, "Expected to get attribute with different value")
		assert.Equal(t, int64(4712), result[0].ID, "Expected to get attribute with different id")
	})

	t.Run("test empty", func(t *testing.T) {
		result, err := sys.GetProjectVersionAttributesByID(777)
		assert.NoError(t, err, "GetProjectVersionAttributesByID call not successful")
		assert.Equal(t, 0, len(result), "Expected to not get any attributes")
	})

	t.Run("test HTTP error", func(t *testing.T) {
		_, err := sys.GetProjectVersionAttributesByID(999)
		assert.Error(t, err, "Expected error but got success")
	})
}

func TestCreateProjectVersion(t *testing.T) {
	// Start a local HTTP server
	sys, server := spinUpServer(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/projectVersions" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			bodyBytes, _ := ioutil.ReadAll(req.Body)
			bodyContent := string(bodyBytes)
			responseContent := `{"data": `
			responseContent += bodyContent
			responseContent += `,"count": 1,"responseCode": 201,"links": {}}`
			fmt.Println(responseContent)
			rw.WriteHeader(201)
			rw.Write([]byte(responseContent))
			return
		}
	})

	// Close the server when test finishes
	defer server.Close()

	t.Run("test success", func(t *testing.T) {
		int64Value := int64(65)
		int32Value := int32(876)
		float32Value := float32(19.12)
		now := models.NewIso8601MilliDateTime()
		enabled := true
		disabled := false
		name := "Test new PV"
		owner := "someUser"
		masterGUID := "dsadaoudoiud"
		project := models.Project{CreatedBy: &owner, CreationDate: now, Description: name, ID: int64Value, IssueTemplateID: &name, Name: &name}
		projectVersionState := models.ProjectVersionState{AnalysisResultsExist: &disabled, AnalysisUploadEnabled: &disabled,
			AttentionRequired: &disabled, AuditEnabled: &enabled, BatchBugSubmissionExists: &disabled, Committed: &enabled,
			CriticalPriorityIssueCountDelta: &int32Value, DeltaPeriod: &int32Value, ExtraMessage: &name, HasCustomIssues: &disabled,
			ID: &int64Value, IssueCountDelta: &int32Value, LastFprUploadDate: &now, MetricEvaluationDate: &now, PercentAuditedDelta: &float32Value,
			PercentCriticalPriorityIssuesAuditedDelta: &float32Value}
		version := models.ProjectVersion{AssignedIssuesCount: int64Value, Project: &project, Name: &name, Active: &enabled,
			Committed: &enabled, AttachmentsOutOfDate: disabled, AutoPredict: disabled, BugTrackerEnabled: &disabled,
			CustomTagValuesAutoApply: disabled, RefreshRequired: disabled, Owner: &owner, ServerVersion: &float32Value,
			SnapshotOutOfDate: &disabled, StaleIssueTemplate: &disabled, MasterAttrGUID: &masterGUID,
			LatestScanID: &int64Value, IssueTemplateName: &name, IssueTemplateModifiedTime: &int64Value,
			IssueTemplateID: &name, Description: &name, CreatedBy: &owner, BugTrackerPluginID: &name, Mode: "NONE",
			CurrentState: &projectVersionState, ID: int64Value, LoadProperties: "", CreationDate: &now,
			MigrationVersion: float32Value, ObfuscatedID: "", PredictionPolicy: "", SecurityGroup: "",
			SiteID: "", SourceBasePath: "", Status: "", TracesOutOfDate: false}
		result, err := sys.CreateProjectVersion(&version)
		assert.NoError(t, err, "CreateProjectVersion call not successful")
		assert.Equal(t, name, *result.Name, "Expected to get PV with different value")
		assert.Equal(t, int64(65), result.ID, "Expected to get PV with different id")
	})
}

func TestProjectVersionCopyFromPartial(t *testing.T) {
	// Start a local HTTP server
	bodyContent := ""
	sys, server := spinUpServer(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/projectVersions/action/copyFromPartial" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			bodyBytes, _ := ioutil.ReadAll(req.Body)
			bodyContent = string(bodyBytes)
			rw.Write([]byte(
				`{"data":[{"latestScanId":null,"serverVersion":17.2,"tracesOutOfDate":false,"attachmentsOutOfDate":false,"description":"",
				"project":{"id":4711,"name":"python-test","description":"","creationDate":"2018-12-03T06:29:38.197+0000","createdBy":"someUser",
				"issueTemplateId":"dasdasdasdsadasdasdasdasdas"},"sourceBasePath":null,"mode":"BASIC","masterAttrGuid":"sddasdasda","obfuscatedId":null,
				"id":10172,"customTagValuesAutoApply":null,"issueTemplateId":"dasdasdasdsadasdasdasdasdas","loadProperties":null,"predictionPolicy":null,
				"bugTrackerPluginId":null,"owner":"admin","_href":"https://fortify.mo.sap.corp/ssc/api/v1/projectVersions/10172",
				"committed":true,"bugTrackerEnabled":false,"active":true,"snapshotOutOfDate":false,"issueTemplateModifiedTime":1578411924701,
				"securityGroup":null,"creationDate":"2018-02-09T16:59:41.297+0000","refreshRequired":false,"issueTemplateName":"someTemplate",
				"migrationVersion":null,"createdBy":"admin","name":"0","siteId":null,"staleIssueTemplate":false,"autoPredict":null,
				"currentState":{"id":10172,"committed":true,"attentionRequired":false,"analysisResultsExist":true,"auditEnabled":true,
				"lastFprUploadDate":"2018-02-09T16:59:53.497+0000","extraMessage":null,"analysisUploadEnabled":true,"batchBugSubmissionExists":false,
				"hasCustomIssues":false,"metricEvaluationDate":"2018-03-10T00:02:45.553+0000","deltaPeriod":7,"issueCountDelta":0,"percentAuditedDelta":0.0,
				"criticalPriorityIssueCountDelta":0,"percentCriticalPriorityIssuesAuditedDelta":0.0},"assignedIssuesCount":0,"status":null}],
				"count":1,"responseCode":200,"links":{"last":{"href":"https://fortify.mo.sap.corp/ssc/api/v1/projects/4711/versions?start=0"},
				"first":{"href":"https://fortify.mo.sap.corp/ssc/api/v1/projects/4711/versions?start=0"}}}`))
			return
		}
	})
	// Close the server when test finishes
	defer server.Close()

	t.Run("test success", func(t *testing.T) {
		expected := `{"copyAnalysisProcessingRules":true,"copyBugTrackerConfiguration":true,"copyCurrentStateFpr":true,"copyCustomTags":true,"previousProjectVersionId":10172,"projectVersionId":10173}
`
		err := sys.ProjectVersionCopyFromPartial(10172, 10173)
		assert.NoError(t, err, "ProjectVersionCopyFromPartial call not successful")
		assert.Equal(t, expected, bodyContent, "Different request content expected")
	})
}

func TestProjectVersionCopyCurrentState(t *testing.T) {
	// Start a local HTTP server
	bodyContent := ""
	sys, server := spinUpServer(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/projectVersions/action/copyCurrentState" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			bodyBytes, _ := ioutil.ReadAll(req.Body)
			bodyContent = string(bodyBytes)
			rw.Write([]byte(
				`{"data":[{"latestScanId":null,"serverVersion":17.2,"tracesOutOfDate":false,"attachmentsOutOfDate":false,"description":"",
				"project":{"id":4711,"name":"python-test","description":"","creationDate":"2018-12-03T06:29:38.197+0000","createdBy":"someUser",
				"issueTemplateId":"dasdasdasdsadasdasdasdasdas"},"sourceBasePath":null,"mode":"BASIC","masterAttrGuid":"sddasdasda","obfuscatedId":null,
				"id":10172,"customTagValuesAutoApply":null,"issueTemplateId":"dasdasdasdsadasdasdasdasdas","loadProperties":null,"predictionPolicy":null,
				"bugTrackerPluginId":null,"owner":"admin","_href":"https://fortify.mo.sap.corp/ssc/api/v1/projectVersions/10172",
				"committed":true,"bugTrackerEnabled":false,"active":true,"snapshotOutOfDate":false,"issueTemplateModifiedTime":1578411924701,
				"securityGroup":null,"creationDate":"2018-02-09T16:59:41.297+0000","refreshRequired":false,"issueTemplateName":"someTemplate",
				"migrationVersion":null,"createdBy":"admin","name":"0","siteId":null,"staleIssueTemplate":false,"autoPredict":null,
				"currentState":{"id":10172,"committed":true,"attentionRequired":false,"analysisResultsExist":true,"auditEnabled":true,
				"lastFprUploadDate":"2018-02-09T16:59:53.497+0000","extraMessage":null,"analysisUploadEnabled":true,"batchBugSubmissionExists":false,
				"hasCustomIssues":false,"metricEvaluationDate":"2018-03-10T00:02:45.553+0000","deltaPeriod":7,"issueCountDelta":0,"percentAuditedDelta":0.0,
				"criticalPriorityIssueCountDelta":0,"percentCriticalPriorityIssuesAuditedDelta":0.0},"assignedIssuesCount":0,"status":null}],
				"count":1,"responseCode":200,"links":{"last":{"href":"https://fortify.mo.sap.corp/ssc/api/v1/projects/4711/versions?start=0"},
				"first":{"href":"https://fortify.mo.sap.corp/ssc/api/v1/projects/4711/versions?start=0"}}}`))
			return
		}
	})
	// Close the server when test finishes
	defer server.Close()

	t.Run("test success", func(t *testing.T) {
		expected := `{"copyCurrentStateFpr":true,"previousProjectVersionId":10172,"projectVersionId":10173}
`
		err := sys.ProjectVersionCopyCurrentState(10172, 10173)
		assert.NoError(t, err, "ProjectVersionCopyCurrentState call not successful")
		assert.Equal(t, expected, bodyContent, "Different request content expected")
	})
}

func TestCopyProjectVersionPermissions(t *testing.T) {
	// Start a local HTTP server
	bodyContent := ""
	referenceContent := `[{"displayName":"some user","email":"some.one@test.com","entityName":"some_user","firstName":"some","id":589,"lastName":"user","type":"User"}]
`
	response := `{"data": `
	response += referenceContent
	response += `,"count": 1,"responseCode": 200}`
	sys, server := spinUpServer(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/projectVersions/10172/authEntities" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			rw.Write([]byte(response))
			return
		}
		if req.URL.Path == "/projectVersions/10173/authEntities" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			bodyBytes, _ := ioutil.ReadAll(req.Body)
			bodyContent = string(bodyBytes)
			rw.Write([]byte(response))
			return
		}
	})
	// Close the server when test finishes
	defer server.Close()

	t.Run("test success", func(t *testing.T) {
		err := sys.CopyProjectVersionPermissions(10172, 10173)
		assert.NoError(t, err, "CopyProjectVersionPermissions call not successful")
		assert.Equal(t, referenceContent, bodyContent, "Different request content expected")
	})
}

func TestCommitProjectVersion(t *testing.T) {
	// Start a local HTTP server
	bodyContent := ""
	referenceContent := `{"active":null,"bugTrackerEnabled":null,"bugTrackerPluginId":null,"committed":true,"createdBy":null,"creationDate":null,"description":null,"issueTemplateId":null,"issueTemplateModifiedTime":null,"issueTemplateName":null,"latestScanId":null,"masterAttrGuid":null,"name":null,"owner":null,"serverVersion":null,"snapshotOutOfDate":null,"staleIssueTemplate":null}
`
	response := `{"data": `
	response += referenceContent
	response += `,"count": 1,"responseCode": 200}`
	sys, server := spinUpServer(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/projectVersions/10172" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			bodyBytes, _ := ioutil.ReadAll(req.Body)
			bodyContent = string(bodyBytes)
			rw.Write([]byte(response))
			return
		}
	})
	// Close the server when test finishes
	defer server.Close()

	t.Run("test success", func(t *testing.T) {
		result, err := sys.CommitProjectVersion(10172)
		assert.NoError(t, err, "CommitProjectVersion call not successful")
		assert.Equal(t, true, *result.Committed, "Different result content expected")
		assert.Equal(t, referenceContent, bodyContent, "Different request content expected")
	})
}

func TestInactivateProjectVersion(t *testing.T) {
	// Start a local HTTP server
	bodyContent := ""
	referenceContent := `{"active":false,"bugTrackerEnabled":null,"bugTrackerPluginId":null,"committed":true,"createdBy":null,"creationDate":null,"description":null,"issueTemplateId":null,"issueTemplateModifiedTime":null,"issueTemplateName":null,"latestScanId":null,"masterAttrGuid":null,"name":null,"owner":null,"serverVersion":null,"snapshotOutOfDate":null,"staleIssueTemplate":null}
`
	response := `{"data": `
	response += referenceContent
	response += `,"count": 1,"responseCode": 200}`
	sys, server := spinUpServer(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/projectVersions/10172" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			bodyBytes, _ := ioutil.ReadAll(req.Body)
			bodyContent = string(bodyBytes)
			rw.Write([]byte(response))
			return
		}
	})
	// Close the server when test finishes
	defer server.Close()

	t.Run("test success", func(t *testing.T) {
		result, err := sys.InactivateProjectVersion(10172)
		assert.NoError(t, err, "InactivateProjectVersion call not successful")
		assert.Equal(t, true, *result.Committed, "Different result content expected")
		assert.Equal(t, false, *result.Active, "Different result content expected")
		assert.Equal(t, referenceContent, bodyContent, "Different request content expected")
	})
}

func TestGetArtifactsOfProjectVersion(t *testing.T) {
	// Start a local HTTP server
	response := `{"data": [{"artifactType": "FPR","fileName": "df54e2ade34c4f6aaddf35679dd87a21.tmp","approvalDate": null,"messageCount": 0,
		"scanErrorsCount": 0,"uploadIP": "10.238.8.48","allowApprove": false,"allowPurge": false,"lastScanDate": "2019-11-26T22:37:52.000+0000",
		"fileURL": null,"id": 56,"purged": false,"webInspectStatus": "NONE","inModifyingStatus": false,"originalFileName": "result.fpr",
		"allowDelete": true,"scaStatus": "PROCESSED","indexed": true,"runtimeStatus": "NONE","userName": "some_user","versionNumber": null,
		"otherStatus": "NOT_EXIST","uploadDate": "2019-11-26T22:38:11.813+0000","approvalComment": null,"approvalUsername": null,"fileSize": 984703,
		"messages": "","auditUpdated": false,"status": "PROCESS_COMPLETE"}],"count": 1,"responseCode": 200}`
	sys, server := spinUpServer(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/projectVersions/10172/artifacts" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			rw.Write([]byte(response))
			return
		}
	})
	// Close the server when test finishes
	defer server.Close()

	t.Run("test success", func(t *testing.T) {
		result, err := sys.GetArtifactsOfProjectVersion(10172)
		assert.NoError(t, err, "GetArtifactsOfProjectVersion call not successful")
		assert.Equal(t, 1, len(result), "Different result content expected")
		assert.Equal(t, int64(56), result[0].ID, "Different result content expected")
	})
}

func TestGetFilterSetOfProjectVersionByTitle(t *testing.T) {
	// Start a local HTTP server
	response := `{"data":[{"defaultFilterSet":true,"folders":[
	{"id":1,"guid":"4711","name":"Corporate Security Requirements","color":"000000"},
	{"id":2,"guid":"4712","name":"Audit All","color":"ff0000"},
	{"id":3,"guid":"4713","name":"Spot Checks of Each Category","color":"ff8000"},
	{"id":4,"guid":"4714","name":"Optional","color":"808080"}],"description":"",
	"guid":"666","title":"Special"}],"count":1,"responseCode":200}}`
	sys, server := spinUpServer(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/projectVersions/10172/filterSets" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			rw.Write([]byte(response))
			return
		}
	})
	// Close the server when test finishes
	defer server.Close()

	t.Run("test success", func(t *testing.T) {
		result, err := sys.GetFilterSetOfProjectVersionByTitle(10172, "Special")
		assert.NoError(t, err, "GetFilterSetOfProjectVersionByTitle call not successful")
		assert.Equal(t, "Special", result.Title, "Different result content expected")
	})

	t.Run("test default", func(t *testing.T) {
		result, err := sys.GetFilterSetOfProjectVersionByTitle(10172, "")
		assert.NoError(t, err, "GetFilterSetOfProjectVersionByTitle call not successful")
		assert.Equal(t, "Special", result.Title, "Different result content expected")
	})
}

func TestGetIssueFilterSelectorOfProjectVersionByName(t *testing.T) {
	// Start a local HTTP server
	response := `{"data":{"groupBySet": [{"entityType": "CUSTOMTAG","guid": "adsffghjkl","displayName": "Analysis",
	"value": "87f2364f-dcd4-49e6-861d-f8d3f351686b","description": ""},{"entityType": "ISSUE","guid": "lkjhgfd",
	"displayName": "Category","value": "11111111-1111-1111-1111-111111111165","description": ""}],"filterBySet":[{
	"entityType": "CUSTOMTAG","filterSelectorType": "LIST","guid": "87f2364f-dcd4-49e6-861d-f8d3f351686b","displayName": "Analysis",
	"value": "87f2364f-dcd4-49e6-861d-f8d3f351686b","description": "The analysis tag must be set.",
	"selectorOptions": []},{"entityType": "FOLDER","filterSelectorType": "LIST","guid": "userAssignment","displayName": "Folder",
	"value": "FOLDER","description": "","selectorOptions": []}]},"responseCode":200}}`
	sys, server := spinUpServer(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/projectVersions/10172/issueSelectorSet" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			rw.Write([]byte(response))
			return
		}
	})
	// Close the server when test finishes
	defer server.Close()

	t.Run("test success one", func(t *testing.T) {
		result, err := sys.GetIssueFilterSelectorOfProjectVersionByName(10172, "Analysis")
		assert.NoError(t, err, "GetIssueFilterSelectorOfProjectVersionByName call not successful")
		assert.NotNil(t, result, "Expected non nil value")
		assert.Equal(t, 1, len(result.FilterBySet), "Different result expected")
		assert.Equal(t, 1, len(result.GroupBySet), "Different result expected")
	})

	t.Run("test success several", func(t *testing.T) {
		result, err := sys.GetIssueFilterSelectorOfProjectVersionByName(10172, "Analysis", "Folder")
		assert.NoError(t, err, "GetIssueFilterSelectorOfProjectVersionByName call not successful")
		assert.NotNil(t, result, "Expected non nil value")
		assert.Equal(t, 2, len(result.FilterBySet), "Different result expected")
		assert.Equal(t, 1, len(result.GroupBySet), "Different result expected")
	})

	t.Run("test empty", func(t *testing.T) {
		result, err := sys.GetIssueFilterSelectorOfProjectVersionByName(10172, "Some", "Other")
		assert.NoError(t, err, "GetIssueFilterSelectorOfProjectVersionByName call not successful")
		assert.NotNil(t, result, "Expected non nil value")
		assert.Equal(t, 0, len(result.FilterBySet), "Different result expected")
		assert.Equal(t, 0, len(result.GroupBySet), "Different result expected")
	})
}

func TestGetProjectIssuesByIDAndFilterSetGroupedByFolder(t *testing.T) {
	// Start a local HTTP server
	sys, server := spinUpServer(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/projectVersions/10172/filterSets" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			rw.Write([]byte(`{"data":[{"defaultFilterSet":true,"folders":[
				{"id":1,"guid":"4711","name":"Corporate Security Requirements","color":"000000"},
				{"id":2,"guid":"4712","name":"Audit All","color":"ff0000"},
				{"id":3,"guid":"4713","name":"Spot Checks of Each Category","color":"ff8000"},
				{"id":4,"guid":"4714","name":"Optional","color":"808080"}],"description":"",
				"guid":"666","title":"Special"}],"count":1,"responseCode":200}}`))
			return
		}
		if req.URL.Path == "/projectVersions/10172/issueSelectorSet" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			rw.Write([]byte(`{"data":{"groupBySet": [{"entityType": "ISSUE","guid": "FOLDER","displayName": "Folder","value": "FOLDER",
			"description": ""}],"filterBySet":[]},"responseCode":200}}`))
			return
		}
		if req.URL.Path == "/projectVersions/10172/issueGroups" {
			assert.Equal(t, "filterset=666&groupingtype=FOLDER&showsuppressed=true", req.URL.RawQuery)
			return
		}
		rw.WriteHeader(400)
	})
	// Close the server when test finishes
	defer server.Close()

	t.Run("test success", func(t *testing.T) {
		_, err := sys.GetProjectIssuesByIDAndFilterSetGroupedByFolder(10172, "Special")
		assert.NoError(t, err, "CopyProjectVersionPermissions call not successful")
	})
}

func TestGetProjectIssuesByIDAndFilterSetGroupedByCategory(t *testing.T) {
	// Start a local HTTP server
	sys, server := spinUpServer(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/projectVersions/10172/filterSets" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			rw.Write([]byte(`{"data":[{"defaultFilterSet":true,"folders":[
				{"id":1,"guid":"4711","name":"Corporate Security Requirements","color":"000000"},
				{"id":2,"guid":"4712","name":"Audit All","color":"ff0000"},
				{"id":3,"guid":"4713","name":"Spot Checks of Each Category","color":"ff8000"},
				{"id":4,"guid":"4714","name":"Optional","color":"808080"}],"description":"",
				"guid":"666","title":"Special"}],"count":1,"responseCode":200}}`))
			return
		}
		if req.URL.Path == "/projectVersions/10172/issueSelectorSet" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			rw.Write([]byte(`{"data":{"groupBySet": [{"entityType": "ISSUE","guid": "11111111-1111-1111-1111-111111111165",
			"displayName": "Category","value": "11111111-1111-1111-1111-111111111165","description": ""}],"filterBySet":[]},"responseCode":200}}`))
			return
		}
		if req.URL.Path == "/projectVersions/10172/issueGroups" {
			assert.Equal(t, "filter=4713&filterset=666&groupingtype=11111111-1111-1111-1111-111111111165&showsuppressed=true", req.URL.RawQuery)
			return
		}
		rw.WriteHeader(400)
	})
	// Close the server when test finishes
	defer server.Close()

	t.Run("test success", func(t *testing.T) {
		_, err := sys.GetProjectIssuesByIDAndFilterSetGroupedByCategory(10172, "Special")
		assert.NoError(t, err, "CopyProjectVersionPermissions call not successful")
	})
}

func TestGetProjectIssuesByIDAndFilterSetGroupedByAnalysis(t *testing.T) {
	// Start a local HTTP server
	sys, server := spinUpServer(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/projectVersions/10172/filterSets" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			rw.Write([]byte(`{"data":[{"defaultFilterSet":true,"folders":[
				{"id":1,"guid":"4711","name":"Corporate Security Requirements","color":"000000"},
				{"id":2,"guid":"4712","name":"Audit All","color":"ff0000"},
				{"id":3,"guid":"4713","name":"Spot Checks of Each Category","color":"ff8000"},
				{"id":4,"guid":"4714","name":"Optional","color":"808080"}],"description":"",
				"guid":"666","title":"Special"}],"count":1,"responseCode":200}}`))
			return
		}
		if req.URL.Path == "/projectVersions/10172/issueSelectorSet" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			rw.Write([]byte(`{"data":{"groupBySet": [{"entityType": "CUSTOMTAG","guid": "87f2364f-dcd4-49e6-861d-f8d3f351686b","displayName": "Analysis",
			"value": "87f2364f-dcd4-49e6-861d-f8d3f351686b","description": ""}],"filterBySet":[]},"responseCode":200}}`))
			return
		}
		if req.URL.Path == "/projectVersions/10172/issueGroups" {
			assert.Equal(t, "filterset=666&groupingtype=87f2364f-dcd4-49e6-861d-f8d3f351686b&showsuppressed=true", req.URL.RawQuery)
			return
		}
		rw.WriteHeader(400)
	})
	// Close the server when test finishes
	defer server.Close()

	t.Run("test success", func(t *testing.T) {
		_, err := sys.GetProjectIssuesByIDAndFilterSetGroupedByAnalysis(10172, "Special")
		assert.NoError(t, err, "CopyProjectVersionPermissions call not successful")
	})
}

func TestGetIssueStatisticsOfProjectVersion(t *testing.T) {
	// Start a local HTTP server
	response := `{"data": [{"filterSetId": 3887,"hiddenCount": 0,"suppressedDisplayableCount": 0,"suppressedCount": 11,"hiddenDisplayableCount": 0,"projectVersionId": 10172,
				"removedDisplayableCount": 0,"removedCount": 747}],"count": 1,"responseCode": 200}`
	sys, server := spinUpServer(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/projectVersions/10172/issueStatistics" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			rw.Write([]byte(response))
			return
		}
	})
	// Close the server when test finishes
	defer server.Close()

	t.Run("test success", func(t *testing.T) {
		result, err := sys.GetIssueStatisticsOfProjectVersion(10172)
		assert.NoError(t, err, "GetArtifactsOfProjectVersion call not successful")
		assert.Equal(t, 1, len(result), "Different result content expected")
		assert.Equal(t, int64(10172), *result[0].ProjectVersionID, "Different result content expected")
		assert.Equal(t, int32(11), *result[0].SuppressedCount, "Different result content expected")
	})
}

func TestGenerateQGateReport(t *testing.T) {
	// Start a local HTTP server
	data := ""
	sys, server := spinUpServer(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/reports" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			bodyBytes, _ := ioutil.ReadAll(req.Body)
			data = string(bodyBytes)
			response := `{"data": `
			response += data
			response += `,"responseCode": 201}`
			rw.WriteHeader(201)
			rw.Write([]byte(response))
			return
		}
	})
	// Close the server when test finishes
	defer server.Close()

	t.Run("test success", func(t *testing.T) {
		result, err := sys.GenerateQGateReport(2837, 17540, "Fortify", "develop", "PDF")
		assert.NoError(t, err, "GetArtifactsOfProjectVersion call not successful")
		assert.Equal(t, int64(2837), result.Projects[0].ID, "Different result content expected")
		assert.Equal(t, int64(17540), result.Projects[0].Versions[0].ID, "Different result content expected")
		assert.Equal(t, int64(18), *result.ReportDefinitionID, "Different result content expected")
	})
}

func TestGetReportDetails(t *testing.T) {
	// Start a local HTTP server
	response := `{"data": {"id":999,"name":"FortifyReport","note":"","type":"PORTFOLIO","reportDefinitionId":18,"format":"PDF",
	"projects":[{"id":2837,"name":"Fortify","versions":[{"id":17540,"name":"develop"}]}],"projectVersionDisplayName":"develop",
	"inputReportParameters":[{"name":"Q-gate-report","identifier":"projectVersionId","paramValue":17540,"type":"SINGLE_PROJECT"}]},"count": 1,"responseCode": 200}`
	sys, server := spinUpServer(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/reports/999" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			rw.Write([]byte(response))
			return
		}
	})
	// Close the server when test finishes
	defer server.Close()

	t.Run("test success", func(t *testing.T) {
		result, err := sys.GetReportDetails(999)
		assert.NoError(t, err, "GetReportDetails call not successful")
		assert.Equal(t, int64(999), result.ID, "Different result content expected")
	})
}

func TestGetFileUploadToken(t *testing.T) {
	// Start a local HTTP server
	bodyContent := ""
	reference := `{"fileTokenType":"UPLOAD"}
`
	response := `{"data": {"fileTokenType": "UPLOAD","token": "ZjE1OTdjZjEtMjAzNS00NTFmLThiOWItNzBkYzI0MWEzZGNj"},"responseCode": 201}`
	sys, server := spinUpServer(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/fileTokens" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			bodyBytes, _ := ioutil.ReadAll(req.Body)
			bodyContent = string(bodyBytes)
			rw.WriteHeader(201)
			rw.Write([]byte(response))
			return
		}
	})
	// Close the server when test finishes
	defer server.Close()

	t.Run("test success", func(t *testing.T) {
		result, err := sys.GetFileUploadToken()
		assert.NoError(t, err, "GetFileUploadToken call not successful")
		assert.Equal(t, "ZjE1OTdjZjEtMjAzNS00NTFmLThiOWItNzBkYzI0MWEzZGNj", result.Token, "Different result content expected")
		assert.Equal(t, reference, bodyContent, "Different request content expected")
	})
}

func TestGetFileDownloadToken(t *testing.T) {
	// Start a local HTTP server
	bodyContent := ""
	reference := `{"fileTokenType":"DOWNLOAD"}
`
	response := `{"data": {"fileTokenType": "DOWNLOAD","token": "ZjE1OTdjZjEtMjAzNS00NTFmLThiOWItNzBkYzI0MWEzZGNj"},"responseCode": 201}`
	sys, server := spinUpServer(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/fileTokens" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			bodyBytes, _ := ioutil.ReadAll(req.Body)
			bodyContent = string(bodyBytes)
			rw.WriteHeader(201)
			rw.Write([]byte(response))
			return
		}
	})
	// Close the server when test finishes
	defer server.Close()

	t.Run("test success", func(t *testing.T) {
		result, err := sys.GetFileDownloadToken()
		assert.NoError(t, err, "GetFileDownloadToken call not successful")
		assert.Equal(t, "ZjE1OTdjZjEtMjAzNS00NTFmLThiOWItNzBkYzI0MWEzZGNj", result.Token, "Different result content expected")
		assert.Equal(t, reference, bodyContent, "Different request content expected")
	})
}

func TestGetReportFileToken(t *testing.T) {
	// Start a local HTTP server
	bodyContent := ""
	reference := `{"fileTokenType":"REPORT_FILE"}
`
	response := `{"data": {"fileTokenType": "REPORT_FILE","token": "ZjE1OTdjZjEtMjAzNS00NTFmLThiOWItNzBkYzI0MWEzZGNj"},"responseCode": 201}`
	sys, server := spinUpServer(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/fileTokens" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			bodyBytes, _ := ioutil.ReadAll(req.Body)
			bodyContent = string(bodyBytes)
			rw.WriteHeader(201)
			rw.Write([]byte(response))
			return
		}
	})
	// Close the server when test finishes
	defer server.Close()

	t.Run("test success", func(t *testing.T) {
		result, err := sys.GetReportFileToken()
		assert.NoError(t, err, "GetReportFileToken call not successful")
		assert.Equal(t, "ZjE1OTdjZjEtMjAzNS00NTFmLThiOWItNzBkYzI0MWEzZGNj", result.Token, "Different result content expected")
		assert.Equal(t, reference, bodyContent, "Different request content expected")
	})
}

func TestInvalidateFileToken(t *testing.T) {
	// Start a local HTTP server
	response := `{"responseCode": 200}`
	sys, server := spinUpServer(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/fileTokens" && req.Method == "DELETE" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			rw.WriteHeader(200)
			rw.Write([]byte(response))
			return
		}
	})
	// Close the server when test finishes
	defer server.Close()

	t.Run("test success", func(t *testing.T) {
		err := sys.InvalidateFileTokens()
		assert.NoError(t, err, "InvalidateFileTokens call not successful")
	})
}

func TestUploadFile(t *testing.T) {
	// Start a local HTTP server
	bodyContent := ""
	getTokenCalled := false
	invalidateTokenCalled := false
	sys, server := spinUpServer(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/fileTokens" && req.Method == "DELETE" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			rw.WriteHeader(200)
			rw.Write([]byte(`{"responseCode": 200}`))
			invalidateTokenCalled = true
			return
		}
		if req.URL.Path == "/fileTokens" && req.Method == "POST" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			rw.WriteHeader(201)
			rw.Write([]byte(`{"data": {"token": "89ee873"}, "responseCode": 201}`))
			getTokenCalled = true
			return
		}
		if req.URL.Path == "/upload/resultFileUpload.html" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			bodyBytes, _ := ioutil.ReadAll(req.Body)
			bodyContent = string(bodyBytes)
			rw.WriteHeader(200)
			rw.Write([]byte("OK"))
			return
		}
	})
	// Close the server when test finishes
	defer server.Close()

	testFile, err := ioutil.TempFile("", "result.fpr")
	if err != nil {
		t.FailNow()
	}
	defer os.RemoveAll(testFile.Name()) // clean up

	t.Run("test success", func(t *testing.T) {
		err := sys.UploadFile("/upload/resultFileUpload.html", testFile.Name(), 10770)
		assert.NoError(t, err, "UploadFile call not successful")
		assert.Contains(t, bodyContent, `Content-Disposition: form-data; name="file"; filename=`, "Expected different content in request body")
		assert.Contains(t, bodyContent, `Content-Disposition: form-data; name="mat"`, "Expected different content in request body")
		assert.Contains(t, bodyContent, `89ee873`, "Expected different content in request body")
		assert.Contains(t, bodyContent, `Content-Disposition: form-data; name="entityId"`, "Expected different content in request body")
		assert.Contains(t, bodyContent, `10770`, "Expected different content in request body")
		assert.Equal(t, true, getTokenCalled, "Expected GetUploadToken to be called")
		assert.Equal(t, true, invalidateTokenCalled, "Expected InvalidateFileTokens to be called")
	})
}

func TestDownloadResultFile(t *testing.T) {
	// Start a local HTTP server
	bodyContent := ""
	getTokenCalled := false
	invalidateTokenCalled := false
	sys, server := spinUpServer(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/fileTokens" && req.Method == "DELETE" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			rw.WriteHeader(200)
			rw.Write([]byte(`{"responseCode": 200}`))
			invalidateTokenCalled = true
			return
		}
		if req.URL.Path == "/fileTokens" && req.Method == "POST" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			rw.WriteHeader(201)
			rw.Write([]byte(`{"data": {"token": "89ee873"}, "responseCode": 201}`))
			getTokenCalled = true
			return
		}
		if req.URL.Path == "/download/currentStateFprDownload.html" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			bodyBytes, _ := ioutil.ReadAll(req.Body)
			bodyContent = string(bodyBytes)
			rw.WriteHeader(200)
			rw.Write([]byte("OK"))
			return
		}
	})
	// Close the server when test finishes
	defer server.Close()

	t.Run("test success", func(t *testing.T) {
		data, err := sys.DownloadResultFile("/download/currentStateFprDownload.html", 10775)
		assert.NoError(t, err, "DownloadResultFile call not successful")
		assert.Equal(t, "id=10775&mat=89ee873", bodyContent, "Expected different request body")
		assert.Equal(t, []byte("OK"), data, "Expected different result")
		assert.Equal(t, true, getTokenCalled, "Expected GetUploadToken to be called")
		assert.Equal(t, true, invalidateTokenCalled, "Expected InvalidateFileTokens to be called")
	})
}

func TestDownloadReportFile(t *testing.T) {
	// Start a local HTTP server
	getTokenCalled := false
	invalidateTokenCalled := false
	sys, server := spinUpServer(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/fileTokens" && req.Method == "DELETE" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			rw.WriteHeader(200)
			rw.Write([]byte(`{"responseCode": 200}`))
			invalidateTokenCalled = true
			return
		}
		if req.URL.Path == "/fileTokens" && req.Method == "POST" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			rw.WriteHeader(201)
			rw.Write([]byte(`{"data": {"token": "89ee873"}, "responseCode": 201}`))
			getTokenCalled = true
			return
		}
		if req.URL.Path == "/transfer/reportDownload.html" && req.URL.RawQuery == "id=10775&mat=89ee873" {
			header := rw.Header()
			header.Add("Content-type", "application/json")
			rw.WriteHeader(200)
			rw.Write([]byte("OK"))
			return
		}
	})
	// Close the server when test finishes
	defer server.Close()

	t.Run("test success", func(t *testing.T) {
		data, err := sys.DownloadReportFile("/transfer/reportDownload.html", 10775)
		assert.NoError(t, err, "DownloadReportFile call not successful")
		assert.Equal(t, []byte("OK"), data, "Expected different result")
		assert.Equal(t, true, getTokenCalled, "Expected GetUploadToken to be called")
		assert.Equal(t, true, invalidateTokenCalled, "Expected InvalidateFileTokens to be called")
	})
}