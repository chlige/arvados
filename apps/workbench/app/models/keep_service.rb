# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class KeepService < ArvadosBase
  def self.creatable?
    false
  end

  #
  # Retrieve status information from a keep disk
  #
  def status
    if self.service_type == "disk"
        clnt = HTTPClient.new
        header = { 'Authorization' => 'OAuth2 ' + Thread.current[:arvados_api_token]}
        url = "http://" + service_host.to_s + ":" + service_port.to_s + "/status.json"
        response = clnt.get(url, nil, header)
        begin
          resp = Oj.load(response.content, :symbol_keys => true)
        rescue Oj::ParseError
          resp = nil
        end
        resp
    end
  end
end
