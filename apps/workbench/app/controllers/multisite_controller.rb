# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class MultisiteController < ApplicationController
  skip_before_filter :find_object_by_uuid
end
